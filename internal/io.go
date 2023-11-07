package internal

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
)

// utils file
var (
	Bigendian    = new(BigEndian)
	Littleendian = new(LittleEndian)
)

type endian interface {
	Uint(src []byte, n uint) uint64
	Write(v uint64, n int, p []byte) error
}

type BigEndian struct {
}
type LittleEndian struct {
}

func (s *BigEndian) Uint(src []byte, n uint) uint64 {
	step := n
	var length uint = uint(len(src))
	if length < step {
		step = length
	}
	var ans uint64
	if step > 0 {
		var i uint
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "error pos", i, "src length", len(src))
				os.Exit(1)
			}
		}()
		for i = 0; i < step; i++ {
			ans *= 256
			ans += uint64(src[i])
		}
	}
	return ans
}
func (s *BigEndian) Write(v uint64, n int, p []byte) error {
	if len(p) < n {
		return Str_Error("p out of range n")
	}
	var i int
	for i < n {
		p[n-1-i] = uint8(v % 256)
		if v > 0 {
			v /= 256
		}
		i++
	}
	return nil
}

func (s *LittleEndian) Uint(src []byte, n uint) uint64 {
	step := n
	var length uint = uint(len(src))
	if length < step {
		step = length
	}
	var ans uint64
	if step > 1 {
		var i int
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "error pos", i, "src length", len(src), "step", step)
				os.Exit(1)
			}
		}()
		for i = int(step - 1); i >= 0; i-- {
			ans *= 256
			ans += uint64(src[i])
		}
	}
	return ans
}
func (s *LittleEndian) Write(v uint64, n int, p []byte) error {
	if len(p) < n {
		return Str_Error("p out of range n")
	}
	var i int
	for i < n {
		p[i] = uint8(v % 256)
		if v > 0 {
			v /= 256
		}
		i++
	}
	return nil
}

type Reader struct {
	cache_buffer []byte
	mux          sync.Mutex
}

func (s *Reader) Read(reader io.Reader, n int, end endian) uint64 {
	s.mux.Lock()
	defer s.mux.Unlock()
	// lang, err := reader.Read(s.cache_buffer[0:n])
	err := read(reader, s.cache_buffer[0:n])
	if err == nil {
		if n == 1 {
			return uint64(s.cache_buffer[0])
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, "[panic error] lang", n, len(s.cache_buffer), r)
				os.Exit(1)
			}
		}()
		return end.Uint(s.cache_buffer[0:n], uint(n))
	}
	return 0
}
func (s *Reader) RawRead(reader io.Reader, n int) []byte {
	s.mux.Lock()
	defer s.mux.Unlock()
	var (
		ans []byte = nil
	)
	err := read(reader, s.cache_buffer[:n])
	if err == nil {
		ans = make([]byte, n)
		copy(ans, s.cache_buffer[:n])
	}
	return ans
}
func NewReader(size uint64) *Reader {
	return &Reader{cache_buffer: make([]byte, size)}
}

type Str_Error string

func (s Str_Error) Error() string {
	return string(s)
}

type ReadWriter int

func (s ReadWriter) Read(p []byte) (n int, err error) {
	return syscall.Read(int(s), p)
}

func (s ReadWriter) Write(p []byte) (n int, err error) {
	return syscall.Write(int(s), p)
}

// read full byte array
func read(in io.Reader, p []byte) error {
	var (
		err   error
		n     int
		start int
		lang  = len(p)
	)
	n, err = in.Read(p[start:])
	for err == nil && start < lang {
		start += n
		if start >= lang {
			break
		}
		n, err = in.Read(p[start:])
		if n == 0 {
			err = io.EOF
		}
	}
	return err
}
