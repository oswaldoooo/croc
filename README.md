# CRPC

### Use go reflect to come true rpc. Based on Tcp.

## Server
**Use Example**
```go
import (
    "github.com/oswaldoooo/crpc"
    "fmt"
    "os"    
)
type Greet struct{}
func (s *Greet)Greet()string{
    return "hello world!"
}
func main(){
    srv,err:=crpc.Listen(&crpc.Addr{Port: 9000})
    if err==nil{
        crpc.Register(&Greet{})//register your type which hold method
        fmt.Fprintln(os.Stderr,srv.Serve())
    }
}
```

## Client
**Use Example**
```go
import(
    "github.com/oswaldoooo/crpc"
    "fmt"
    "os"
)
func main(){
    cli, err := crpc.Dial(&crpc.Addr{Port: 8000, IP: crpc.IP{127, 0, 0, 1}})
	if err == nil {
		dt, err := cli.Call("Greet", "Greet")
        fmt.Println(dt,err)
	} else {
		fmt.Fprintln(os.Stderr, err.Error())
	}
}
```