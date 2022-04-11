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
	Lst    net.Listener
	Rmt    string
	Itrcpt interceptor
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

func Init(localAddr, localPort, remoteAddr, remotePort string, itr interceptor) (*Proxy, error) {
	p := Proxy{}
	p.Rmt = fmt.Sprintf("%s:%s", remoteAddr, remotePort)

	var err error
	lcl := fmt.Sprintf("%s:%s", localAddr, localPort)
	p.Lst, err = net.Listen("tcp", lcl)
	if err != nil {
		return nil, errors.Wrap(err, errListener)
	}

	p.Itrcpt = itr

	return &p, nil
}

func (p *Proxy) ListenAndServe(bfsz int) {
	for {
		c, err := p.Lst.Accept()
		if err != nil {
			panic(err)
		}
		p.handleConn(c, bfsz)
		c.Close()
	}
}

func (p *Proxy) handleConn(src net.Conn, bfsz int) {
	dst, err := net.Dial("tcp", p.Rmt)
	if err != nil {
		fmt.Println(errRemoteConn)
		return
	}
	defer dst.Close()

	errc := make(chan error, 1)
	go p.copyData(errc, src, dst, bfsz)
	go p.copyData(errc, dst, src, bfsz)
	switch <-errc {
	case nil:
		return
	default:
		fmt.Printf("%v, %s\n", time.Now(), <-errc)
	}
}

func (p *Proxy) copyData(errc chan<- error, src, dst net.Conn, bfsz int) {
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
	bb := make([]byte, bfsz)
	for {
		nr, er := src.Read(bb)
		if nr > 0 {
			bb, ie := p.Itrcpt(bb, nr)
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
