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

// FileSystem provides the routines to allow the static HTTP and TFTP services to render
// templates on demand..
type FileSystem struct {
	sync.Mutex
	lower    string
	logger   *log.Logger
	dynamics map[string]*renderedTemplate
}

// NewFS creates a new initialized filesystem that will fall back to
// serving files from backingFSPath if there is not a template to be
// rendered.
func NewFS(backingFSPath string, logger *log.Logger) *FileSystem {
	return &FileSystem{
		lower:    backingFSPath,
		logger:   logger,
		dynamics: map[string]*renderedTemplate{},
	}
}

// Open tests for the existence of a template to be rendered for a
// file read request.  The returned Reader contains the rendered
// template if there is one, and the returned error contains any
// rendering errors.  If both the reader and error are nil, there is
// no template to be expanded for p and FileSystem should fall back to
// serving a static file.
func (fs *FileSystem) Open(p string, remoteIP net.IP) (*bytes.Reader, error) {
	p = path.Clean(p)
	fs.Lock()
	res, ok := fs.dynamics[p]
	fs.Unlock()
	if !ok {
		return nil, nil
	}
	res.Vars.Lock()
	defer res.Vars.Unlock()
	res.Vars.remoteIP = remoteIP
	return res.write()
}

// ServeHTTP implements http.Handler for the FileSystem.
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

// TftpResponder returns a function that allows the TFTP midlayer to
// serve files from the FileSystem.
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
