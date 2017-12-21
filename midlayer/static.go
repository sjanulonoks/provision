package midlayer

import (
	"net"
	"net/http"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/backend"
)

func ServeStatic(listenAt string, responder http.Handler, logger logger.Logger, pubs *backend.Publishers) (*http.Server, error) {
	conn, err := net.Listen("tcp", listenAt)
	if err != nil {
		return nil, err
	}
	svr := &http.Server{
		Addr:    listenAt,
		Handler: responder,
		ConnState: func(n net.Conn, cs http.ConnState) {
			laddr, lok := n.LocalAddr().(*net.TCPAddr)
			raddr, rok := n.RemoteAddr().(*net.TCPAddr)
			if lok && rok && cs == http.StateActive {
				backend.AddToCache(laddr.IP, raddr.IP)
			}
			return
		},
	}
	go func() {
		if err := svr.Serve(conn); err != nil {
			if err != http.ErrServerClosed {
				logger.Fatalf("Static HTTP server error %v", err)
			}
		}
	}()
	return svr, nil
}
