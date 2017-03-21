package frontend

import (
	"log"
	"net"
	"net/http"
)

func ServeStatic(listenAt, fsPath string) error {
	conn, err := net.Listen("tcp", listenAt)
	if err != nil {
		return err
	}
	fs := http.FileServer(http.Dir(fsPath))
	http.Handle("/", fs)
	go func() {
		if err := http.Serve(conn, nil); err != nil {
			log.Fatalf("Static HTTP server error %v", err)
		}
	}()
	return nil
}
