package midlayer

import (
	"log"
	"net"
	"net/http"

	"github.com/rackn/rocket-skates/backend"
)

func ServeStatic(listenAt, fsPath string, logger *log.Logger) error {
	conn, err := net.Listen("tcp", listenAt)
	if err != nil {
		return err
	}
	fs := http.FileServer(http.Dir(fsPath))
	svr := &http.Server{
		Addr:    listenAt,
		Handler: fs,
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
			logger.Fatalf("Static HTTP server error %v", err)
		}
	}()
	return nil
}
