package midlayer

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pin/tftp"
)

func ServeTftp(listen, fileRoot string, logger *log.Logger) error {
	a, err := net.ResolveUDPAddr("udp", listen)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return err
	}
	svr := tftp.NewServer(func(filename string, rf io.ReaderFrom) error {
		p := filepath.Join(fileRoot, filename)
		p = filepath.Clean(p)
		if !strings.HasPrefix(p, fileRoot+string(filepath.Separator)) {
			err := fmt.Errorf("Filename %s tries to escape root %s", filename, fileRoot)
			logger.Println(err)
			return err
		}
		logger.Printf("Sending %s from %s", filename, p)
		file, err := os.Open(p)
		if err != nil {
			logger.Println(err)
			return err
		}
		if t, ok := rf.(tftp.OutgoingTransfer); ok {
			local := t.LocalAddr()
			remote := t.RemoteAddr()
			addToCache(local, remote.IP)
			// Need to add a function to add to the remote -> local IP cache
			if fi, err := file.Stat(); err == nil {
				t.SetSize(fi.Size())
			}
		}
		n, err := rf.ReadFrom(file)
		if err != nil {
			logger.Println(err)
			return err
		}
		logger.Printf("%d bytes sent\n", n)
		return nil
	}, nil)

	go svr.Serve(conn)
	return nil
}
