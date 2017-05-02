package backend

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type renderer struct {
	path  string
	write func(net.IP) (*bytes.Reader, error)
}

func (r renderer) register(fs *FileSystem) {
	fs.addDynamic(r.path, r.write)
}

func (r renderer) deregister(fs *FileSystem) {
	fs.delDynamic(r.path)
}

type renderers []renderer

func (r renderers) register(fs *FileSystem) {
	if r == nil || len(r) == 0 {
		return
	}
	for _, rt := range r {
		rt.register(fs)
	}
}

func (r renderers) deregister(fs *FileSystem) {
	if r == nil || len(r) == 0 {
		return
	}
	for _, rt := range r {
		rt.deregister(fs)
	}
}

func newRenderedTemplate(r *RenderData,
	tmplKey,
	path string) renderer {
	p := r.p
	var mKey, eKey string
	eKey = r.Env.Key()
	if r.Machine != nil {
		mKey = r.Machine.Key()
	}
	return renderer{
		path: path,
		write: func(remoteIP net.IP) (*bytes.Reader, error) {
			objs, unlocker := p.lockEnts("machines", "bootenvs")
			defer unlocker()
			var rd *RenderData
			var bootenv *BootEnv
			var machine *Machine
			bidx, bfound := objs[1].find(eKey)
			if !bfound {
				return nil, fmt.Errorf("Bootenv %s has vanished!", eKey)
			}
			bootenv = AsBootEnv(objs[1].d[bidx])
			if !bootenv.OnlyUnknown {
				midx, mfound := objs[0].find(mKey)
				if !mfound {
					return nil, fmt.Errorf("Machine %s has vanished!", mKey)
				}
				machine = AsMachine(objs[0].d[midx])
			}
			rd = newRenderData(p, machine, bootenv)
			rd.remoteIP = remoteIP
			buf := bytes.Buffer{}
			tmpl := bootenv.rootTemplate.Lookup(tmplKey)
			if err := tmpl.Execute(&buf, rd); err != nil {
				return nil, err
			}
			return bytes.NewReader(buf.Bytes()), nil
		},
	}
}

type rMachine struct {
	*Machine
	renderData *RenderData
}

func (n *rMachine) Url() string {
	return n.p.FileURL(n.renderData.remoteIP) + "/" + n.Path()
}

type rBootEnv struct {
	*BootEnv
	renderData *RenderData
}

// PathFor expands the partial paths for kernels and initrds into full
// paths appropriate for specific protocols.
//
// proto can be one of 3 choices:
//    http: Will expand to the URL the file can be accessed over.
//    tftp: Will expand to the path the file can be accessed at via TFTP.
//    disk: Will expand to the path of the file inside the provisioner container.
func (b *rBootEnv) PathFor(proto, f string) string {
	tail := b.pathFor(f)
	switch proto {
	case "tftp":
		return tail
	case "http":
		return b.p.FileURL(b.renderData.remoteIP) + "/" + tail
	default:
		b.p.Logger.Fatalf("Unknown protocol %v", proto)
	}
	return ""
}

func (b *rBootEnv) InstallUrl() string {
	return b.p.FileURL(b.renderData.remoteIP) + "/" + path.Join(b.OS.Name, "install")
}

// JoinInitrds joins the fully expanded initrd paths into a comma-separated string.
func (b *rBootEnv) JoinInitrds(proto string) string {
	fullInitrds := make([]string, len(b.Initrds))
	for i, initrd := range b.Initrds {
		fullInitrds[i] = b.PathFor(proto, initrd)
	}
	return strings.Join(fullInitrds, " ")
}

// RenderData is the struct that is passed to templates as a source of
// parameters and useful methods.
type RenderData struct {
	Machine  *rMachine // The Machine that the template is being rendered for.
	Env      *rBootEnv // The boot environment that provided the template.
	p        *DataTracker
	remoteIP net.IP
}

func newRenderData(p *DataTracker, m *Machine, e *BootEnv) *RenderData {
	res := &RenderData{p: p}
	if m != nil {
		res.Machine = &rMachine{Machine: m, renderData: res}
	}
	if e != nil {
		res.Env = &rBootEnv{BootEnv: e, renderData: res}
	}
	return res
}

func (r *RenderData) ProvisionerAddress() string {
	return r.p.LocalIP(r.remoteIP)
}

func (r *RenderData) ProvisionerURL() string {
	return r.p.FileURL(r.remoteIP)
}

func (r *RenderData) ApiURL() string {
	return r.p.ApiURL(r.remoteIP)
}

func (r *RenderData) GenerateToken() string {
	var t string
	if r.Machine == nil {
		ttl := 600
		if sttl, e := r.p.Pref("unknownTokenTimeout"); e == nil {
			ttl, _ = strconv.Atoi(sttl)
		}
		t, _ = NewClaim("general", ttl).Add("machines", "post", "*").
			Add("machines", "get", "*").Seal(r.p.tokenManager)
	} else {
		ttl := 3600
		if sttl, e := r.p.Pref("knownTokenTimeout"); e == nil {
			ttl, _ = strconv.Atoi(sttl)
		}
		t, _ = NewClaim(r.Machine.Key(), ttl).Add("machines", "patch", r.Machine.Key()).
			Add("machines", "get", r.Machine.Key()).Seal(r.p.tokenManager)
	}
	return t
}

// BootParams is a helper function that expands the BootParams
// template from the boot environment.
func (r *RenderData) BootParams() (string, error) {
	if r.Env == nil {
		return "", fmt.Errorf("Missing bootenv")
	}
	res := &bytes.Buffer{}
	if r.Env.bootParamsTmpl == nil {
		return "", nil
	}
	if err := r.Env.bootParamsTmpl.Execute(res, r); err != nil {
		return "", err
	}
	return res.String(), nil
}

func (r *RenderData) ParseUrl(segment, rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	switch segment {
	case "scheme":
		return parsedUrl.Scheme, nil
	case "host":
		return parsedUrl.Host, nil
	case "path":
		return parsedUrl.Path, nil
	}
	return "", fmt.Errorf("No idea how to get URL part %s from %s", segment, rawUrl)
}

// ParamExists is a helper function for determining the existence of a machine parameter.
func (r *RenderData) ParamExists(key string) bool {
	if r.Machine != nil {
		_, ok := r.Machine.GetParam(key, true)
		if ok {
			return ok
		}
	}
	if o, found := r.p.fetchOne(r.p.NewProfile(), r.p.globalProfileName); found {
		p := AsProfile(o)
		if _, ok := p.Params[key]; ok {
			return true
		}
	}
	return false
}

// Param is a helper function for extracting a parameter from Machine.Params
func (r *RenderData) Param(key string) (interface{}, error) {
	if r.Machine != nil {
		v, ok := r.Machine.GetParam(key, true)
		if ok {
			return v, nil
		}
	}
	if o, found := r.p.fetchOne(r.p.NewProfile(), r.p.globalProfileName); found {
		p := AsProfile(o)
		if v, ok := p.Params[key]; ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("No such machine parameter %s", key)
}
