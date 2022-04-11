package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

// Takes TCP packet content and number of bytes read.
type interceptor func([]byte, int) ([]byte, error)

type Proxy struct {
	lst    net.Listener
	rmt    string
	itcptr interceptor
	bfsz   int
}

const (
	errListener   = "unable to listen"
	errLocalConn  = "unable to establish local connection"
	errRemoteConn = "unable to establish remote connection"
)

var (
	errInvalidWrite = errors.New("invalid write result")
	errShortWrite   = errors.New("short write")
)

func Init(laddr, lport, raddr, rport string, bfsz int, itr interceptor) (*Proxy, error) {
	p := Proxy{}
	p.rmt = fmt.Sprintf("%s:%s", raddr, rport)

	var err error
	lcl := fmt.Sprintf("%s:%s", laddr, lport)
	p.lst, err = net.Listen("tcp", lcl)
	if err != nil {
		return nil, errors.Wrap(err, errListener)
	}

	p.itcptr = itr
	p.bfsz = bfsz

	return &p, nil
}

func (p *Proxy) ListenAndServe() {
	for {
		c, err := p.lst.Accept()
		if err != nil {
			panic(err)
		}
		p.handleConn(c)
		c.Close()
	}
}

func (p *Proxy) handleConn(src net.Conn) {
	dst, err := net.Dial("tcp", p.rmt)
	if err != nil {
		fmt.Println(errRemoteConn)
		return
	}
	defer dst.Close()

	errc := make(chan error, 1)
	go p.copyData(errc, src, dst)
	go p.copyData(errc, dst, src)
	switch <-errc {
	case nil:
		return
	default:
		fmt.Printf("%v, %s\n", time.Now(), <-errc)
	}
}

func (p *Proxy) copyData(errc chan<- error, src, dst net.Conn) {
	buf := bufio.NewReader(src)

	var peeked []byte
	if n := buf.Buffered(); n > 0 {
		fmt.Println(n)
		peeked, _ = buf.Peek(n)
	}

	if len(peeked) > 0 {
		if _, err := dst.Write(peeked); err != nil {
			errc <- err
			return
		}
	}

	var err error
	bb := make([]byte, p.bfsz)
	for {
		nr, er := src.Read(bb)
		if nr > 0 {
			bb, ie := p.itcptr(bb, nr)
			if ie != nil {
				err = ie
				break
			}
			nw, ew := dst.Write(bb[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
				if ew != nil {
					err = ew
					break
				}
				if nr != nw {
					err = errShortWrite
					break
				}
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	errc <- err
}
