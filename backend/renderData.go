package backend

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
)

// RenderTemplate is the result of rendering a BootEnv template
type renderedTemplate struct {
	// Path is the absolute path that the Template will be rendered to.
	path string
	// Template is the template that will rendered
	tmpl string
	// Vars holds the variables that will be used during template expansion.
	Vars *RenderData
}

func (r *renderedTemplate) write() (*bytes.Reader, error) {
	buf := bytes.Buffer{}
	tmpl := r.Vars.Env.rootTemplate.Lookup(r.tmpl)
	if err := tmpl.Execute(&buf, r.Vars); err != nil {
		return nil, err
	}
	return bytes.NewReader(buf.Bytes()), nil
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
	sync.Mutex
	Machine           *rMachine // The Machine that the template is being rendered for.
	Env               *rBootEnv // The boot environment that provided the template.
	renderedTemplates []renderedTemplate
	p                 *DataTracker
	remoteIP          net.IP
}

func (p *DataTracker) NewRenderData(m *Machine, e *BootEnv) *RenderData {
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

func (r *RenderData) render(e *Error) {
	r.Lock()
	defer r.Unlock()
	var missingParams []string
	if len(r.Env.RequiredParams) > 0 && r.Machine == nil {
		e.Errorf("Machine is nil or does not have params")
		return
	}
	for _, param := range r.Env.RequiredParams {
		if !r.ParamExists(param) {
			missingParams = append(missingParams, param)
		}
	}
	if len(missingParams) > 0 {
		e.Errorf("missing required machine params for %s:\n %v", r.Machine.Name, missingParams)
		return
	}
	r.renderedTemplates = make([]renderedTemplate, len(r.Env.Templates))

	for i := range r.Env.Templates {
		ti := &r.Env.Templates[i]
		rt := renderedTemplate{tmpl: ti.id(), Vars: r}

		// first, render the path
		buf := &bytes.Buffer{}
		if err := ti.pathTmpl.Execute(buf, r); err != nil {
			e.Errorf("Error rendering template %s path %s: %v",
				ti.Name,
				ti.Path,
				err)
		} else {
			rt.path = path.Clean("/" + buf.String())
		}
		r.renderedTemplates[i] = rt
		r.p.FS.addDynamic(rt.path, &rt)
	}

}

func (r *RenderData) remove(e *Error) {
	r.Lock()
	defer r.Unlock()
	for _, rt := range r.renderedTemplates {
		r.p.FS.delDynamic(rt.path)
	}
}
