package midlayer

import (
	"bytes"
	"io"
	"log"
	"net"
	"os"

	"github.com/pin/tftp"
	"github.com/rackn/rocket-skates/backend"
)

func ServeTftp(listen string, responder func(string, net.IP) (io.Reader, error), logger *log.Logger) error {
	a, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}
	svr := tftp.NewServer(func(filename string, rf io.ReaderFrom) error {
		var local net.IP
		var remote net.UDPAddr
		t, outgoing := rf.(tftp.OutgoingTransfer)
		if outgoing {
			local = t.LocalIP()
			if local != nil && !local.IsUnspecified() {
				remote = t.RemoteAddr()
				backend.AddToCache(local, remote.IP)
			}
		}
		source, err := responder(filename, remote.IP)
		if err != nil {
			return err
		}
		if outgoing {
			var size int64
			switch src := source.(type) {
			case *os.File:
				defer src.Close()
				if fi, err := src.Stat(); err == nil {
					size = fi.Size()
				}
			case *bytes.Reader:
				size = src.Size()
			}
			t.SetSize(size)
		}
		_, err = rf.ReadFrom(source)
		if err != nil {
			logger.Println(err)
			return err
		}
		return nil
	}, nil)

	go svr.Serve(conn)
	return nil
}
