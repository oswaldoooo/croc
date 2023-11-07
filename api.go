package crpc

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"reflect"
	"syscall"

	"github.com/oswaldoooo/crpc/internal"
)

const ( //signal
	success        = 0x01
	serviceinvaild = 0x02
	methodinvaild  = 0x03
	argsinvaild    = 0x04
)

var (
	debuglog *log.Logger = log.New(os.Stdout, "", log.Lshortfile)
	gbuffer  []byte      = make([]byte, 1<<15)
	//encode and decode function for raw bytes
	DeCode func([]byte, any) error   = json.Unmarshal
	EnCode func(any) ([]byte, error) = json.Marshal
)

/*
message made: status code(1 byte)+data_length(2byte)+content
*/
type ServerMux struct {
	rw io.ReadWriter
}
type ClientMux struct {
	rw io.ReadWriter
}
type Client struct {
	con    net.Conn
	buffer []byte
	r      *internal.Reader
}
type IP [4]byte
type Addr struct {
	IP   IP
	Port int
}
type Server struct {
	fd, eid   int
	quene_len int
}

func Listen(addr *Addr) (svr *Server, err error) {
	sid, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if sid > 0 {
		if err = syscall.Bind(sid, &syscall.SockaddrInet4{Addr: addr.IP, Port: addr.Port}); err == nil {
			err = syscall.Listen(sid, 0)
			if err == nil {
				svr = &Server{fd: sid, quene_len: 10}
			}
		}
		if err != nil {
			syscall.Close(sid)
		}
	} else {
		err = internal.Str_Error("socket create failed")
	}

	return
}
func (s *Server) Serve() error {
	var (
		err error
		eid int
	)
	// defer fmt.Println("error is ", err)
	eid, err = syscall.EpollCreate(s.quene_len)
	if err == nil {
		s.eid = eid
		defer s.Close()
		err = syscall.EpollCtl(eid, syscall.EPOLL_CTL_ADD, s.fd, &syscall.EpollEvent{Fd: int32(s.fd), Events: syscall.EPOLLIN})
		if err == nil {
			var (
				epoll_quene         []syscall.EpollEvent = make([]syscall.EpollEvent, s.quene_len)
				quene_len, i, conid int
				servermux           *ServerMux = new(ServerMux)
			)
			//fmt.Println("into cicyle")
			for {
				quene_len, err = syscall.EpollWait(s.eid, epoll_quene, -1)
				if err == nil {
					if quene_len > 0 {
						for i = 0; i < quene_len; i++ {
							if epoll_quene[i].Fd == int32(s.fd) {
								//accept message
								conid, _, err = syscall.Accept(s.fd)
								if err == nil {
									err = syscall.EpollCtl(eid, syscall.EPOLL_CTL_ADD, conid, &syscall.EpollEvent{Fd: int32(conid), Events: syscall.EPOLLIN})
									if err != nil {
										syscall.Close(conid)
										//debuglog.Println("register connection id to epoll failed")
									}
								} else {
									//debuglog.Println("accept connection failed", err.Error())
								}
							} else {
								servermux.rw = internal.ReadWriter(int(epoll_quene[i].Fd))
								err = servermux.Server()
								if err != nil {
									var str_err internal.Str_Error
									//debuglog.Println("server error", err.Error())
									err = syscall.EpollCtl(eid, syscall.EPOLL_CTL_DEL, int(epoll_quene[i].Fd), nil)
									if err != nil {
										//debuglog.Println("deregister connection failed", err.Error())
										str_err += internal.Str_Error(err.Error())
									}
									err = syscall.Close(int(epoll_quene[i].Fd))
									if err != nil {
										//debuglog.Println("close connection failed", err.Error())
										str_err += internal.Str_Error(err.Error())
									}
									if len(str_err) > 0 {
										err = str_err
									}
								}
							}
						}
					}
				} else {
					break
				}
			}
		}
	}

	return err
}
func (s *Server) Close() error {
	var (
		err  error
		serr internal.Str_Error
	)
	err = syscall.Close(s.fd)
	if err != nil {
		serr += internal.Str_Error(err.Error())
	}
	err = syscall.Close(s.eid)
	if err != nil {
		serr += internal.Str_Error(err.Error())
	}
	if len(serr) > 0 {
		err = serr
	}
	return err
}

func (s *ServerMux) Server() error {
	var (
		err error
		n   int
		v   map[string]any = make(map[string]any)
	)
	n, err = s.rw.Read(gbuffer)

	if err == nil && n > 0 {
		err = DeCode(gbuffer[:n], &v)
		if err == nil {
			//debuglog.Println(v)
			var code uint8 = success
			v, err = internal.Call(v) //if error is not nil,send err msg to caller
			if err != nil {
				switch err {
				case internal.METHOD_NOT_FIND, internal.METHOD_NOT_SET:
					code = methodinvaild
				case internal.SERVICE_NOT_REGISTER, internal.SERVICE_NOT_SET:
					code = serviceinvaild
				case internal.ARGS_ERROR:
					code = argsinvaild
				}
				//debuglog.Println("[error]", err.Error())
				err = nil
			}
			var content []byte
			if len(v) > 0 { //if there is not any data,don't need send message to caller
				content, err = EnCode(v)
			}
			if err == nil {
				//debuglog.Println("write to client", code, string(content))
				err = Write(s.rw, code, content)
			}
			//debuglog.Println("err", err)
		}
	}
	if n == 0 {
		err = io.EOF
	}
	return err
}

func register_newconn(conid, eid int) error {
	var err error
	err = syscall.EpollCtl(eid, syscall.EPOLL_CTL_ADD, conid, &syscall.EpollEvent{Fd: int32(conid), Events: syscall.EPOLLIN})
	return err
}

//client

func Dial(addr *Addr) (cm *Client, err error) {
	var con *net.TCPConn
	con, err = net.DialTCP("tcp", nil, &net.TCPAddr{IP: addr.IP[:], Port: addr.Port})
	if err == nil {
		cm = &Client{con: con, buffer: make([]byte, 1<<10), r: internal.NewReader(1 << 10)}
	}
	return
}

type Data map[string]any
type Args struct {
	Key string
	Val any
}

func (s *Client) Call(servicename, methodname string, args ...Args) (data Data, err error) {
	data = make(Data)
	input := Data{"service": servicename, "method": methodname}
	var ok bool
	for _, ele := range args {
		if _, ok = input[ele.Key]; ok {
			err = internal.Str_Error(ele.Key + "duplicate error")
			return
		} else {
			input[ele.Key] = ele.Val
		}
	}
	var content []byte
	content, err = EnCode(input)
	if err == nil {
		// var n int
		var data_len uint64
		_, err = s.con.Write(content)
		if err == nil {
			switch s.r.Read(s.con, 1, internal.Bigendian) {
			case success:
				data_len = s.r.Read(s.con, 2, internal.Bigendian)
				if data_len > 0 {
					//read data and decode
					rawbytes := s.r.RawRead(s.con, int(data_len))
					if rawbytes != nil {
						err = DeCode(rawbytes, &data)
					}
				}
			case serviceinvaild:
				err = internal.Str_Error("service " + servicename + " is invaild")
			case methodinvaild:
				err = internal.Str_Error("method " + methodname + " is invaild")
			}
		}
		if err != nil {
			return
		}
	}
	return
}

// data length can't over than 64k
func Write(w io.Writer, code uint8, content []byte) error {
	newcontent := make([]byte, len(content)+3)
	newcontent[0] = code
	internal.Bigendian.Write(uint64(len(content)), 2, newcontent[1:3])
	if len(content) > 0 {
		copy(newcontent[3:], content)
	}
	//debuglog.Println("prepare write to client")
	_, err := w.Write(newcontent)
	return err
}

func Register(v any) {
	tp := reflect.TypeOf(v)
	if len(tp.Name()) == 0 {
		tp = tp.Elem()
		if len(tp.Name()) == 0 {
			panic("cant get interface name")
		}
	}
	internal.Register(tp.Name(), v)
}
