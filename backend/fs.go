package backend

import (
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

type FileSystem struct {
	sync.Mutex
	lower    string
	logger   *log.Logger
	dynamics map[string]*renderedTemplate
}

func NewFS(backingFSPath string, logger *log.Logger) *FileSystem {
	return &FileSystem{
		lower:    backingFSPath,
		logger:   logger,
		dynamics: map[string]*renderedTemplate{},
	}
}

func (fs *FileSystem) Open(p string, remoteIP net.IP) (*bytes.Reader, error) {
	p = path.Clean(p)
	fs.Lock()
	res, ok := fs.dynamics[p]
	fs.Unlock()
	if !ok {
		return nil, nil
	}
	res.Vars.remoteIP = remoteIP
	return res.write()
}

func (fs *FileSystem) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/") {
		p = path.Clean("/" + p)
		r.URL.Path = p
	}
	r.Body.Close()
	var raddr net.IP
	raddrStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fs.logger.Printf("Static FS: Failed to resolve %s: %v", r.RemoteAddr, err)
	} else {
		raddr = net.ParseIP(raddrStr)
	}
	out, err := fs.Open(p, raddr)
	if err != nil {
		fs.logger.Printf("Static FS: Failed to render template for %s: %v", p, err)
		w.WriteHeader(http.StatusInternalServerError)
	} else if out != nil {
		w.Header().Set("Content-Length", strconv.FormatInt(out.Size(), 10))
		io.Copy(w, out)
	} else {
		http.ServeFile(w, r, path.Join(fs.lower, p))
	}
}

func (fs *FileSystem) TftpResponder() func(string, net.IP) (io.Reader, error) {
	return func(toSend string, remoteIP net.IP) (io.Reader, error) {
		p := path.Clean("/" + toSend)
		out, err := fs.Open(p, remoteIP)
		if err != nil {
			fs.logger.Printf("Static FS: Failed to render template for %s: %v", p, err)
			return nil, err
		}
		if out != nil {
			return out, nil
		}
		return os.Open(path.Join(fs.lower, p))
	}
}

func (fs *FileSystem) addDynamic(path string, t *renderedTemplate) {
	fs.Lock()
	fs.dynamics[path] = t
	fs.Unlock()
}

func (fs *FileSystem) delDynamic(path string) {
	fs.Lock()
	delete(fs.dynamics, path)
	fs.Unlock()
}
