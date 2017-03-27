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
	http.Handle("/", fs)
	go func() {
		if err := http.Serve(conn, nil); err != nil {
			logger.Fatalf("Static HTTP server error %v", err)
		}
	}()
	return nil
}
