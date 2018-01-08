package backend

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/VictorLowther/jsonpatch2/utils"
	"github.com/digitalrebar/provision/models"
)

type Sizer interface {
	Size() int64
}

type ReadSizer interface {
	io.Reader
	Sizer
}

type renderer struct {
	path, name string
	write      func(net.IP) (io.Reader, error)
}

func (r renderer) register(fs *FileSystem) {
	fs.AddDynamicFile(r.path, r.write)
}

func (r renderer) deregister(fs *FileSystem) {
	fs.DelDynamicFile(r.path)
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
	var prefixes, keys []string
	if r.Task != nil {
		prefixes = append(prefixes, "tasks")
		keys = append(keys, r.Task.Key())
	}
	if r.Stage != nil {
		prefixes = append(prefixes, "stages")
		keys = append(keys, r.Stage.Key())
	}
	if r.Env != nil {
		prefixes = append(prefixes, "bootenvs")
		keys = append(keys, r.Env.Key())
	}
	if r.Machine != nil {
		prefixes = append(prefixes, "machines")
		keys = append(keys, r.Machine.Key())
	}
	targetPrefix := r.target.Prefix()
	dt := r.rt.dt
	return renderer{
		path: path,
		name: tmplKey,
		write: func(remoteIP net.IP) (io.Reader, error) {
			var err error
			rt := dt.Request(r.rt.Logger.Switch("bootenv"),
				"templates",
				"tasks",
				"stages",
				"bootenvs",
				"machines",
				"profiles",
				"params",
				"preferences")
			rd := &RenderData{rt: rt}
			rd.rt.Do(func(d Stores) {
				for i, prefix := range prefixes {
					item := rd.rt.Find(prefix, keys[i])
					if item == nil {
						err = fmt.Errorf("%s:%s has vanished", prefix, keys[i])
					}
					switch obj := item.(type) {
					case *Task:
						rd.Task = &rTask{Task: obj, renderData: rd}
					case *Stage:
						rd.Stage = &rStage{Stage: obj, renderData: rd}
					case *BootEnv:
						rd.Env = &rBootEnv{BootEnv: obj, renderData: rd}
					case *Machine:
						rd.Machine = &rMachine{Machine: obj, renderData: rd}
					default:
						rd.rt.Panicf("%s:%s is neither Renderable nor a machine", item.Prefix(), item.Key())
					}
				}
			})
			if err != nil {
				return nil, err
			}
			switch targetPrefix {
			case "tasks":
				rd.target = renderable(rd.Task.Task)
			case "stages":
				rd.target = renderable(rd.Stage.Stage)
			case "bootenvs":
				rd.target = renderable(rd.Env.BootEnv)
			}
			rd.remoteIP = remoteIP
			buf := bytes.Buffer{}
			tmpl := rd.target.templates().Lookup(tmplKey)
			rd.rt.Do(func(d Stores) {
				err = tmpl.Execute(&buf, rd)
			})
			if err != nil {
				return nil, err
			}
			rd.rt.Debugf("Content:\n%s\n", string(buf.Bytes()))
			return bytes.NewReader(buf.Bytes()), nil
		},
	}
}

type rMachine struct {
	*Machine
	renderData *RenderData
}

func (n *rMachine) Url() string {
	return n.renderData.rt.FileURL(n.renderData.remoteIP) + "/" + n.Path()
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
		return strings.TrimPrefix(tail, "/")
	case "http":
		return b.renderData.rt.FileURL(b.renderData.remoteIP) + tail
	default:
		b.renderData.rt.Fatalf("Unknown protocol %v", proto)
	}
	return ""
}

type Repo struct {
	Tag            string   `json:"tag"`
	OS             []string `json:"os"`
	URL            string   `json:"url"`
	PackageType    string   `json:"packageType"`
	RepoType       string   `json:"repoType"`
	InstallSource  bool     `json:"installSource"`
	SecuritySource bool     `json:"securitySource"`
	Distribution   string   `json:"distribution"`
	Components     []string `json:"components"`
	r              *RenderData
	targetOS       string
}

func (rd *Repo) JoinedComponents() string {
	return strings.Join(rd.Components, " ")
}

func (rd *Repo) R() *RenderData {
	return rd.r
}

func (rd *Repo) Target() string {
	return rd.targetOS
}

func (rd *Repo) osParts() (string, string) {
	parts := strings.SplitN(rd.targetOS, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return rd.targetOS, ""
}

func (rd *Repo) renderStyle() string {
	if rd.RepoType != "" {
		return rd.RepoType
	}
	osName, _ := rd.osParts()
	switch osName {
	case "redhat", "rhel", "centos", "scientificlinux":
		return "yum"
	case "suse", "sles", "opensuse":
		return "zypp"
	case "debian", "ubuntu":
		return "apt"
	default:
		return "unknown"
	}
}

func (rd *Repo) UrlFor(component string) string {
	if rd.InstallSource || rd.Distribution == "" {
		return rd.URL
	}
	osName, _ := rd.osParts()
	switch osName {
	case "centos":
		return fmt.Sprintf("%s/%s/%s/$basearch", rd.URL, rd.Distribution, component)
	case "scientificlinux":
		return fmt.Sprintf("%s/%s/$basearch/%s", rd.URL, rd.Distribution, component)
	default:
		return rd.URL
	}
}

func (rd *Repo) Install() (string, error) {
	tmpl := template.New("installLines").Option("missingkey=error")
	var err error
	switch rd.renderStyle() {
	case "yum":
		if rd.InstallSource {
			tmpl, err = tmpl.Parse(`install
url --url {{.URL}}
repo --name="{{.Tag}}" --baseurl={{.URL}} --cost=100{{if .R.ParamExists "proxy-servers"}} --proxy="{{index (.R.Param "proxy-servers") 0}}"{{end}}`)
		} else {
			tmpl, err = tmpl.Parse(`
repo --name="{{.Tag}}" --baseurl={{.URL}} --cost=100{{if .R.ParamExists "proxy-servers"}} --proxy="{{index (.R.Param "proxy-servers") 0}}"{{end}}`)
		}
	case "apt":
		if rd.InstallSource {
			tmpl, err = tmpl.Parse(`d-i mirror/protocol string {{.R.ParseUrl "scheme" .URL}}
d-i mirror/http/hostname string {{.R.ParseUrl "host" .URL}}
d-i mirror/http/directory string {{.R.ParseUrl "path" .URL}}
`)
		} else {
			tmpl, err = tmpl.Parse(`{{if (eq "debian" .R.Env.OS.Family)}}
d-i apt-setup/security_host string {{.URL}}
{{else}}
d-i apt-setup/security_host string {{.R.ParseUrl "host" .URL}}
d-i apt-setup/security_path string {{.R.ParseUrl "path" .URL}}
{{end}}`)
		}
	default:
		return "", fmt.Errorf("No idea how to handle repos for %s", rd.targetOS)
	}
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, rd)
	return buf.String(), err
}

func (rd *Repo) Lines() (string, error) {
	tmpl := template.New("installLines").Option("missingkey=error")
	var err error
	switch rd.renderStyle() {
	case "yum":
		tmpl, err = tmpl.Parse(`{{range $component := $.Components}}
[{{$.Tag}}-{{$component}}]
name={{$.Tag}} - {{$component}}
baseurl={{$.UrlFor $component}}
gpgcheck=0
{{else}}
[{{$.Tag}}]
name={{$.Target}} - {{$.Tag}}
baseurl={{$.UrlFor ""}}
gpgcheck=0
{{ end }}`)
	case "apt":
		tmpl, err = tmpl.Parse(`deb {{.URL}} {{.Distribution}} {{.JoinedComponents}}
`)
	default:
		return "", fmt.Errorf("No idea how to handle repos for %s", rd.targetOS)
	}
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, rd)
	return buf.String(), err
}

func (b *rBootEnv) InstallUrl() (string, error) {
	repos := b.renderData.InstallRepos()
	if len(repos) == 0 {
		return "", fmt.Errorf("No install repository available")
	}
	return repos[0].URL, nil
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
	rt       *RequestTracker
	target   renderable
	remoteIP net.IP
}

func (r *RenderData) fetchRepos(test func(*Repo) bool) (res []*Repo) {
	res = []*Repo{}
	p, err := r.Param("package-repositories")
	if p == nil || err != nil {
		return
	}
	repos := []*Repo{}
	if utils.Remarshal(p, &repos) != nil {
		return
	}
	for _, repo := range repos {
		if !test(repo) {
			continue
		}
		repo.r = r
		res = append(res, repo)
	}
	return
}

func (r *RenderData) Repos(tags ...string) []*Repo {
	return r.fetchRepos(func(rd *Repo) bool {
		for _, t := range tags {
			if t == rd.Tag {
				return true
			}
		}
		return false
	})
}

func (r *RenderData) MachineRepos() []*Repo {
	found := r.fetchRepos(func(rd *Repo) bool {
		for _, os := range rd.OS {
			if os == r.Machine.OS {
				rd.targetOS = r.Machine.OS
				return true
			}
		}
		return false
	})
	if len(found) == 0 {
		// See if we have something locally available
		for _, obj := range r.rt.d("bootenvs").Items() {
			env := obj.(*BootEnv)
			if env.OS.Name == r.Machine.OS {
				fi, err := os.Stat(path.Join(r.rt.dt.FileRoot, r.Machine.OS, "install", env.Kernel))
				if err == nil && fi.Mode().IsRegular() {
					found = append(found, &Repo{
						Tag:           env.Name,
						InstallSource: true,
						OS:            []string{r.Machine.OS},
						URL:           r.rt.FileURL(r.remoteIP) + "/" + path.Join(r.Machine.OS, "install"),
						r:             r,
						targetOS:      r.Machine.OS,
					})
					break
				}
			}
		}
	}
	return found
}

func (r *RenderData) InstallRepos() []*Repo {
	found := r.MachineRepos()
	var installRepo, updateRepo *Repo
	res := []*Repo{}
	for _, repo := range found {
		if installRepo == nil && repo.InstallSource {
			installRepo = repo
		}
		if updateRepo == nil && repo.SecuritySource {
			updateRepo = repo
		}
	}
	res = append(res, installRepo)
	if updateRepo != nil {
		res = append(res, updateRepo)
	}
	return res
}

func newRenderData(rt *RequestTracker, m *Machine, r renderable) *RenderData {
	res := &RenderData{rt: rt}
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
	if m != nil {
		if res.Env == nil {
			obj := rt.Find("bootenvs", m.BootEnv)
			if obj != nil {
				res.Env = &rBootEnv{BootEnv: obj.(*BootEnv), renderData: res}
			}
		}
		if res.Stage == nil {
			obj := rt.Find("stages", m.Stage)
			if obj != nil {
				res.Stage = &rStage{Stage: obj.(*Stage), renderData: res}
			}
		}
	}
	return res
}

func (r *RenderData) ProvisionerAddress() string {
	return r.rt.dt.LocalIP(r.remoteIP)
}

func (r *RenderData) ProvisionerURL() string {
	return r.rt.FileURL(r.remoteIP)
}

func (r *RenderData) ApiURL() string {
	return r.rt.ApiURL(r.remoteIP)
}

func (r *RenderData) GenerateToken() string {
	var t string

	grantor := "system"
	grantorSecret := ""
	if ss := r.rt.dt.pref("systemGrantorSecret"); ss != "" {
		grantorSecret = ss
	}

	if r.Machine == nil {
		ttl := time.Minute * 10
		if sttl := r.rt.dt.pref("unknownTokenTimeout"); sttl != "" {
			mttl, _ := strconv.Atoi(sttl)
			ttl = time.Second * time.Duration(mttl)
		}
		t, _ = NewClaim("general", grantor, ttl).
			Add("machines", "post", "*").
			Add("machines", "get", "*").
			AddSecrets("", grantorSecret, "").
			Seal(r.rt.dt.tokenManager)
	} else {
		ttl := time.Hour
		if sttl := r.rt.dt.pref("knownTokenTimeout"); sttl != "" {
			mttl, _ := strconv.Atoi(sttl)
			ttl = time.Second * time.Duration(mttl)
		}
		t, _ = NewClaim(r.Machine.Key(), grantor, ttl).
			Add("machines", "*", r.Machine.Key()).
			Add("stages", "get", "*").
			Add("jobs", "create", r.Machine.Key()).
			Add("jobs", "get", r.Machine.Key()).
			Add("jobs", "patch", r.Machine.Key()).
			Add("jobs", "update", r.Machine.Key()).
			Add("jobs", "actions", r.Machine.Key()).
			Add("jobs", "log", r.Machine.Key()).
			Add("tasks", "get", "*").
			Add("info", "get", "*").
			Add("events", "post", "*").
			Add("reservations", "create", "*").
			AddMachine(r.Machine.Key()).
			AddSecrets("", grantorSecret, r.Machine.Secret).
			Seal(r.rt.dt.tokenManager)
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
	if ss := r.rt.dt.pref("systemGrantorSecret"); ss != "" {
		grantorSecret = ss
	}

	ttl := time.Hour * 24 * 7 * 52 * 3
	t, _ := NewClaim(r.Machine.Key(), grantor, ttl).
		Add("machines", "*", r.Machine.Key()).
		Add("stages", "get", "*").
		Add("jobs", "create", r.Machine.Key()).
		Add("jobs", "get", r.Machine.Key()).
		Add("jobs", "patch", r.Machine.Key()).
		Add("jobs", "update", r.Machine.Key()).
		Add("jobs", "actions", r.Machine.Key()).
		Add("jobs", "log", r.Machine.Key()).
		Add("tasks", "get", "*").
		Add("info", "get", "*").
		Add("events", "post", "*").
		Add("reservations", "create", "*").
		AddMachine(r.Machine.Key()).
		AddSecrets("", grantorSecret, r.Machine.Secret).
		Seal(r.rt.dt.tokenManager)
	return t
}

func (r *RenderData) GenerateProfileToken(profile string, duration int) string {
	if r.Machine == nil {
		// Don't allow profile tokens.
		return "UnknownMachineTokenNotAllowed"
	}

	if !r.Machine.HasProfile(profile) {
		// Don't allow profile tokens.
		return "InvalidTokenNotAllowedNotOnMachine"
	}

	if p := r.rt.Find("profiles", profile); p == nil {
		// Don't allow profile tokens.
		return "InvalidTokenNotAllowedNoProfile"
	}

	grantor := "system"
	grantorSecret := ""
	if ss := r.rt.dt.pref("systemGrantorSecret"); ss != "" {
		grantorSecret = ss
	}

	if duration <= 0 {
		duration = 2000000000
	}
	ttl := time.Second * time.Duration(duration)

	t, _ := NewClaim(r.Machine.Key(), grantor, ttl).
		Add("profiles", "get", profile).
		Add("profiles", "update", profile).
		Add("profiles", "patch", profile).
		AddMachine(r.Machine.Key()).
		AddSecrets("", grantorSecret, r.Machine.Secret).
		Seal(r.rt.dt.tokenManager)
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

// Param is a helper function for extracting a parameter from Machine.Params
func (r *RenderData) Param(key string) (interface{}, error) {
	if r.Machine != nil {
		v, ok := r.rt.GetParam(r.Machine, key, true)
		if ok {
			return v, nil
		}
	}
	if o := r.rt.Find("profiles", r.rt.dt.GlobalProfileName); o != nil {
		p := AsProfile(o)
		if v, ok := r.rt.GetParam(p, key, true); ok {
			return v, nil
		}
	}
	return nil, fmt.Errorf("No such machine parameter %s", key)
}

// ParamExists is a helper function for determining the existence of a machine parameter.
func (r *RenderData) ParamExists(key string) bool {
	_, err := r.Param(key)
	return err == nil
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
