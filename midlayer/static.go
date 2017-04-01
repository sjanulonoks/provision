package midlayer

import (
	"log"
	"net"
	"net/http"
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
			//logger.Printf("Static conn: laddr %v raddr %v state %s", n.LocalAddr(), n.RemoteAddr(), cs.String())
			// Need to add function to add an entry to the remote -> local addr cache
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
