package main

import (
	"flag"

	"github.com/michalslomczynski/proxy/proxy"
)

var laddr = flag.String("local address", "0.0.0.0", "local IP address")
var lport = flag.String("local port", "9999", "local Port")
var raddr = flag.String("remote address", "", "remote IP address")
var rport = flag.String("remote port", "", "remote port")
var bfsz = flag.Int("buffer size", 24*1024, "max expected size of TCP packet or more")

func main() {
	flag.Parse()

	itcptr := func([]byte, int) ([]byte, error) {
		// Implement me.
		return nil, nil
	}

	px, err := proxy.Init(*laddr, *lport, *raddr, *rport, *bfsz, itcptr)
	if err != nil {
		panic(err)
	}
	px.ListenAndServe()
}
