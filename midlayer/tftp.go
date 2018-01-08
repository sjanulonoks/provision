package midlayer

import (
	"context"
	"io"
	"net"
	"os"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/backend"
	"github.com/pin/tftp"
)

type TftpHandler struct {
	srv *tftp.Server
}

func (h *TftpHandler) Shutdown(ctx context.Context) error {
	h.srv.Shutdown()
	return nil
}

func ServeTftp(listen string, responder func(string, net.IP) (io.Reader, error),
	log logger.Logger, pubs *backend.Publishers) (Service, error) {
	a, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, err
	}
	readHandler := func(filename string, rf io.ReaderFrom) error {
		var local net.IP
		var remote net.UDPAddr
		t, outgoing := rf.(tftp.OutgoingTransfer)
		rpi, haveRPI := rf.(tftp.RequestPacketInfo)
		if outgoing && haveRPI {
			local = rpi.LocalIP()
		}
		if outgoing {
			remote = t.RemoteAddr()
		}
		if outgoing && haveRPI {
			backend.AddToCache(local, remote.IP)
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
			case backend.Sizer:
				size = src.Size()
			}
			t.SetSize(size)
		}
		_, err = rf.ReadFrom(source)
		if err != nil {
			log.Errorf("TFTP transfer error: %v", err)
			return err
		}
		return nil
	}
	svr := tftp.NewServer(readHandler, nil)

	th := &TftpHandler{srv: svr}

	go svr.Serve(conn)

	return th, nil
}
