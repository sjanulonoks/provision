package backend

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/digitalrebar/provision/models"
)

type renderer struct {
	path, name string
	write      func(net.IP) (*bytes.Reader, error)
}

func (r renderer) register(fs *FileSystem) {
	fs.addDynamic(r.path, r.write)
}

func (r renderer) deregister(fs *FileSystem) {
	fs.delDynamic(r.path)
}

type renderers []renderer

type renderable interface {
	models.Model
	renderInfo() ([]models.TemplateInfo, []string)
	templates() *template.Template
}

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
	var prefixes, keys []string
	if r.Task != nil {
		prefixes = []string{"tasks", "machines"}
		keys = []string{r.target.Key(), r.Machine.Key()}
	} else if r.Machine == nil {
		prefixes = []string{"bootenvs"}
		keys = []string{r.target.Key()}
	} else if r.Stage != nil {
		prefixes = []string{"machines", "stages"}
		keys = []string{r.Machine.Key(), r.target.Key()}
	} else {
		prefixes = []string{"machines", "bootenvs"}
		keys = []string{r.Machine.Key(), r.target.Key()}
	}
	return renderer{
		path: path,
		name: tmplKey,
		write: func(remoteIP net.IP) (*bytes.Reader, error) {
			objs, unlocker := p.LockEnts("stages", "tasks", "machines", "bootenvs", "profiles")
			defer unlocker()
			var rd *RenderData
			var machine *Machine
			var target renderable
			for i, prefix := range prefixes {
				item := objs(prefix).Find(keys[i])
				if item == nil {
					return nil, fmt.Errorf("%s:%s has vanished", prefix, keys[i])
				}
				switch item.(type) {
				case renderable:
					target = item.(renderable)
				case *Machine:
					machine = AsMachine(item)
				default:
					p.Logger.Panicf("%s:%s is neither Renderable nor a machine", item.Prefix(), item.Key())
				}
			}
			rd = newRenderData(objs, p, machine, target)
			rd.remoteIP = remoteIP
			buf := bytes.Buffer{}
			tmpl := target.templates().Lookup(tmplKey)
			if err := tmpl.Execute(&buf, rd); err != nil {
				return nil, err
			}
			p.Debugf("debugRenderer", "Content:\n%s\n", string(buf.Bytes()))
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

type rTask struct {
	*Task
	renderData *RenderData
}

type rStage struct {
	*Stage
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
	Task     *rTask
	Stage    *rStage
	d        Stores
	target   renderable
	p        *DataTracker
	remoteIP net.IP
}

func newRenderData(d Stores, p *DataTracker, m *Machine, r renderable) *RenderData {
	res := &RenderData{d: d, p: p}
	res.target = r
	if m != nil {
		res.Machine = &rMachine{Machine: m, renderData: res}
	}
	switch obj := r.(type) {
	case *BootEnv:
		res.Env = &rBootEnv{BootEnv: obj, renderData: res}
	case *Task:
		res.Task = &rTask{Task: obj, renderData: res}
	case *Stage:
		res.Stage = &rStage{Stage: obj, renderData: res}
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

	grantor := "system"
	grantorSecret := ""
	if ss := r.p.pref("systemGrantorSecret"); ss != "" {
		grantorSecret = ss
	}

	if r.Machine == nil {
		ttl := time.Minute * 10
		if sttl := r.p.pref("unknownTokenTimeout"); sttl != "" {
			mttl, _ := strconv.Atoi(sttl)
			ttl = time.Second * time.Duration(mttl)
		}
		t, _ = NewClaim("general", grantor, ttl).
			Add("machines", "post", "*").
			Add("machines", "get", "*").
			AddSecrets("", grantorSecret, "").
			Seal(r.p.tokenManager)
	} else {
		ttl := time.Hour
		if sttl := r.p.pref("knownTokenTimeout"); sttl != "" {
			mttl, _ := strconv.Atoi(sttl)
			ttl = time.Second * time.Duration(mttl)
		}
		t, _ = NewClaim(r.Machine.Key(), grantor, ttl).
			Add("machines", "*", r.Machine.Key()).
			Add("stages", "get", "*").
			Add("jobs", "create", r.Machine.Key()).
			Add("jobs", "get", r.Machine.Key()).
			Add("jobs", "patch", r.Machine.Key()).
			Add("jobs", "actions", r.Machine.Key()).
			Add("jobs", "log", r.Machine.Key()).
			Add("tasks", "get", "*").
			Add("info", "get", "*").
			Add("events", "post", "*").
			Add("reservations", "create", "*").
			AddMachine(r.Machine.Key()).
			AddSecrets("", grantorSecret, r.Machine.Secret).
			Seal(r.p.tokenManager)
	}
	return t
}

func (r *RenderData) GenerateInfiniteToken() string {
	if r.Machine == nil {
		// Don't allow infinite tokens.
		return ""
	}

	grantor := "system"
	grantorSecret := ""
	if ss := r.p.pref("systemGrantorSecret"); ss != "" {
		grantorSecret = ss
	}

	ttl := time.Hour * 24 * 7 * 52 * 3
	t, _ := NewClaim(r.Machine.Key(), grantor, ttl).
		Add("machines", "*", r.Machine.Key()).
		Add("stages", "get", "*").
		Add("jobs", "create", r.Machine.Key()).
		Add("jobs", "get", r.Machine.Key()).
		Add("jobs", "patch", r.Machine.Key()).
		Add("jobs", "actions", r.Machine.Key()).
		Add("jobs", "log", r.Machine.Key()).
		Add("tasks", "get", "*").
		Add("info", "get", "*").
		Add("events", "post", "*").
		Add("reservations", "create", "*").
		AddMachine(r.Machine.Key()).
		AddSecrets("", grantorSecret, r.Machine.Secret).
		Seal(r.p.tokenManager)
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
		_, ok := r.Machine.GetParam(r.d, key, true)
		if ok {
			return ok
		}
	}
	if o := r.d("profiles").Find(r.p.GlobalProfileName); o != nil {
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
		v, ok := r.Machine.GetParam(r.d, key, true)
		if ok {
			return v, nil
		}
	}
	if o := r.d("profiles").Find(r.p.GlobalProfileName); o != nil {
		p := AsProfile(o)
		if v, ok := p.Params[key]; ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("No such machine parameter %s", key)
}

func (r *RenderData) CallTemplate(name string, data interface{}) (ret interface{}, err error) {
	buf := bytes.NewBuffer([]byte{})
	tmpl := r.target.templates().Lookup(name)
	if tmpl == nil {
		return nil, fmt.Errorf("Missing template: %s", name)
	}
	err = tmpl.Execute(buf, data)
	if err == nil {
		ret = buf.String()
	}
	return
}

func (r *RenderData) makeRenderers(e models.ErrorAdder) renderers {
	toRender, requiredParams := r.target.renderInfo()
	for _, param := range requiredParams {
		if !r.ParamExists(param) {
			e.Errorf("Missing required parameter %s for %s %s", param, r.target.Prefix(), r.target.Key())
		}
	}
	rts := make(renderers, len(toRender))
	for i := range toRender {
		tmplPath := ""
		ti := &toRender[i]
		if ti.PathTemplate() != nil {
			// first, render the path
			buf := &bytes.Buffer{}
			if err := ti.PathTemplate().Execute(buf, r); err != nil {
				e.Errorf("Error rendering template %s path %s: %v",
					ti.Name,
					ti.Path,
					err)
				continue
			}
			if r.target.Prefix() == "tasks" {
				tmplPath = path.Clean(buf.String())
			} else {
				tmplPath = path.Clean("/" + buf.String())
			}
		}
		rts[i] = newRenderedTemplate(r, ti.Id(), tmplPath)
	}
	return renderers(rts)
}
