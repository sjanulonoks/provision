package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// BootEnv encapsulates the machine-agnostic information needed by the
// provisioner to set up a boot environment.
//
// swagger:model
type BootEnv struct {
	*models.BootEnv
	validate
	renderers      renderers
	bootParamsTmpl *template.Template
	p              *DataTracker
	rootTemplate   *template.Template
	tmplMux        sync.Mutex
}

func (obj *BootEnv) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *BootEnv) SaveClean() store.KeySaver {
	mod := *obj.BootEnv
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func (b *BootEnv) HasTask(s string) bool {
	for _, p := range b.Tasks {
		if p == s {
			return true
		}
	}
	return false
}

func (b *BootEnv) Indexes() map[string]index.Maker {
	fix := AsBootEnv
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Name": index.Make(
			true,
			"string",
			func(i, j models.Model) bool {
				return fix(i).Name < fix(j).Name
			},
			func(ref models.Model) (gte, gt index.Test) {
				name := fix(ref).Name
				return func(s models.Model) bool {
						return fix(s).Name >= name
					},
					func(s models.Model) bool {
						return fix(s).Name > name
					}
			},
			func(s string) (models.Model, error) {
				res := fix(b.New())
				res.Name = s
				return res, nil
			}),
		"Available": index.Make(
			false,
			"boolean",
			func(i, j models.Model) bool {
				return (!fix(i).Available) && fix(j).Available
			},
			func(ref models.Model) (gte, gt index.Test) {
				avail := fix(ref).Available
				return func(s models.Model) bool {
						v := fix(s).Available
						return v || (v == avail)
					},
					func(s models.Model) bool {
						return fix(s).Available && !avail
					}
			},
			func(s string) (models.Model, error) {
				res := fix(b.New())
				switch s {
				case "true":
					res.Available = true
				case "false":
					res.Available = false
				default:
					return nil, errors.New("Available must be true or false")
				}
				return res, nil
			}),
		"OnlyUnknown": index.Make(
			false,
			"boolean",
			func(i, j models.Model) bool {
				return !fix(i).OnlyUnknown && fix(j).OnlyUnknown
			},
			func(ref models.Model) (gte, gt index.Test) {
				unknown := fix(ref).OnlyUnknown
				return func(s models.Model) bool {
						v := fix(s).OnlyUnknown
						return v || (v == unknown)
					},
					func(s models.Model) bool {
						return fix(s).OnlyUnknown && !unknown
					}
			},
			func(s string) (models.Model, error) {
				res := fix(b.New())
				switch s {
				case "true":
					res.OnlyUnknown = true
				case "false":
					res.OnlyUnknown = false
				default:
					return nil, errors.New("OnlyUnknown must be true or false")
				}
				return res, nil
			}),
	}
}

func (b *BootEnv) Backend() store.Store {
	return b.p.getBackend(b)
}

func (b *BootEnv) pathFor(f string) string {
	res := b.OS.Name
	if strings.HasSuffix(b.Name, "-install") {
		res = path.Join(res, "install")
	}
	return path.Clean(path.Join(res, f))
}

func (b *BootEnv) localPathFor(f string) string {
	return path.Join(b.p.FileRoot, b.pathFor(f))
}

func (b *BootEnv) genRoot(commonRoot *template.Template, e models.ErrorAdder) *template.Template {
	res := models.MergeTemplates(commonRoot, b.Templates, e)
	for i, tmpl := range b.Templates {
		if tmpl.Path == "" {
			e.Errorf("Template[%d] needs a Path", i)
		}
	}
	if b.BootParams != "" {
		tmpl, err := template.New("machine").Parse(b.BootParams)
		if err != nil {
			e.Errorf("Error compiling boot parameter template: %v", err)
		} else {
			b.bootParamsTmpl = tmpl.Option("missingkey=error")
		}
	}
	if b.HasError() != nil {
		return nil
	}
	return res
}

func (b *BootEnv) explodeIso() {
	// Only work on things that are requested.
	if b.OS.IsoFile == "" {
		b.p.Infof("debugBootEnv", "Explode ISO: Skipping %s becausing no iso image specified\n", b.Name)
		return
	}
	// Have we already exploded this?  If file exists, then good!
	canaryPath := b.localPathFor("." + b.OS.Name + ".rebar_canary")
	buf, err := ioutil.ReadFile(canaryPath)
	if err == nil && len(buf) != 0 && string(bytes.TrimSpace(buf)) == b.OS.IsoSha256 {
		b.p.Infof("debugBootEnv", "Explode ISO: canary file %s, in place and has proper SHA256\n", canaryPath)
		return
	}

	isoPath := filepath.Join(b.p.FileRoot, "isos", b.OS.IsoFile)
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		b.Errorf("Explode ISO: iso doesn't exist: %s\n", isoPath)
		if b.OS.IsoUrl != "" {
			b.Errorf("You can download the required ISO from %s", b.OS.IsoUrl)
		}
		return
	}

	// Only check the has if we have one.
	if b.OS.IsoSha256 != "" {
		f, err := os.Open(isoPath)
		if err != nil {
			b.Errorf("Explode ISO: failed to open iso file %s: %v", isoPath, err)
			return
		}
		defer f.Close()
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			b.Errorf("Explode ISO: failed to read iso file %s: %v", isoPath, err)
			return
		}
		hash := hex.EncodeToString(hasher.Sum(nil))
		if hash != b.OS.IsoSha256 {
			b.Errorf("Explode ISO: SHA256 bad. actual: %v expected: %v", hash, b.OS.IsoSha256)
			return
		}
	}

	// Call extract script
	// /explode_iso.sh b.OS.Name fileRoot isoPath path.Dir(canaryPath)
	cmdName := path.Join(b.p.FileRoot, "explode_iso.sh")
	cmdArgs := []string{b.OS.Name, b.p.FileRoot, isoPath, b.localPathFor(""), b.OS.IsoSha256}
	if out, err := exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		b.Errorf("Explode ISO: explode_iso.sh failed for %s: %s", b.Name, err)
		b.Errorf("Command output:\n%s", string(out))

	} else {
		b.p.Infof("debugBootEnv", "Explode ISO: %s exploded to %s", b.OS.IsoFile, isoPath)
		b.p.Debugf("debugBootEnv", "Explode ISO Log:\n%s", string(out))
	}
	return
}

func (b *BootEnv) Validate() {
	b.renderers = renderers{}
	// First, the stuff that must be correct in order for
	b.AddError(index.CheckUnique(b, b.stores("bootenvs").Items()))
	for _, taskName := range b.Tasks {
		if b.stores("tasks").Find(taskName) == nil {
			b.Errorf("Task %s does not exist", taskName)
		}
	}
	// If our basic templates do not parse, it is game over for us
	b.p.tmplMux.Lock()
	b.tmplMux.Lock()
	root := b.genRoot(b.p.rootTemplate, b)
	b.p.tmplMux.Unlock()
	if root != nil {
		b.rootTemplate = root
	}
	b.tmplMux.Unlock()
	if !b.SetValid() {
		// If we have not been validated at this point, return.
		return
	}
	// OK, we are sane, if not useable.  Check to see if we are useable
	seenPxeLinux := false
	seenELilo := false
	seenIPXE := false
	for _, template := range b.Templates {
		if template.Name == "pxelinux" {
			seenPxeLinux = true
		}
		if template.Name == "elilo" {
			seenELilo = true
		}
		if template.Name == "ipxe" {
			seenIPXE = true
		}
	}
	if !seenIPXE {
		if !(seenPxeLinux && seenELilo) {
			b.Errorf("bootenv: Missing elilo or pxelinux template")
		}
	}
	// Make sure the ISO for this bootenv has been exploded locally so that
	// the boot env can use its contents.
	if b.OS.IsoFile != "" {
		b.explodeIso()
	}
	// If we have a non-empty Kernel, make sure it points at something kernel-ish.
	if b.Kernel != "" {
		kPath := b.localPathFor(b.Kernel)
		kernelStat, err := os.Stat(kPath)
		if err != nil {
			b.Errorf("bootenv: %s: missing kernel %s (%s)",
				b.Name,
				b.Kernel,
				kPath)
		} else if !kernelStat.Mode().IsRegular() {
			b.Errorf("bootenv: %s: invalid kernel %s (%s)",
				b.Name,
				b.Kernel,
				kPath)
		}
	}
	// Ditto for all the initrds.
	if len(b.Initrds) > 0 {
		for _, initrd := range b.Initrds {
			iPath := b.localPathFor(initrd)
			initrdStat, err := os.Stat(iPath)
			if err != nil {
				b.Errorf("bootenv: %s: missing initrd %s (%s)",
					b.Name,
					initrd,
					iPath)
				continue
			}
			if !initrdStat.Mode().IsRegular() {
				b.Errorf("bootenv: %s: invalid initrd %s (%s)",
					b.Name,
					initrd,
					iPath)
			}
		}
	}
	if b.OnlyUnknown {
		b.renderers = append(b.renderers, b.Render(b.stores, nil, b)...)
	} else {
		machines := b.stores("machines")
		if machines != nil {
			for _, i := range machines.Items() {
				machine := AsMachine(i)
				if machine.BootEnv != b.Name {
					continue
				}
				b.renderers = append(b.renderers, b.Render(b.stores, machine, b)...)
			}
		}
	}
	b.SetAvailable()
}

func (b *BootEnv) OnLoad() error {
	b.stores = func(ref string) *Store {
		return b.p.objs[ref]
	}
	defer func() { b.stores = nil }()
	return b.BeforeSave()
}

func (b *BootEnv) New() store.KeySaver {
	res := &BootEnv{BootEnv: &models.BootEnv{}}
	if b.BootEnv != nil && b.ChangeForced() {
		res.ForceChange()
	}
	res.p = b.p
	return res
}

func (b *BootEnv) setDT(p *DataTracker) {
	b.p = p
}

func (b *BootEnv) BeforeSave() error {
	b.Validate()
	if !b.Validated {
		return b.MakeError(422, ValidationError, b)
	}
	return nil
}

func (b *BootEnv) BeforeDelete() error {
	e := &models.Error{Code: 409, Type: StillInUseError, Object: b}
	machines := b.stores("machines")
	prefToFind := ""
	if b.OnlyUnknown {
		prefToFind = "unknownBootEnv"
	} else {
		prefToFind = "defaultBootEnv"
	}
	if b.p.pref(prefToFind) == b.Name {
		e.Errorf("BootEnv %s is the active %s, cannot remove it", b.Name, prefToFind)
	}
	if !b.OnlyUnknown {
		for _, i := range machines.Items() {
			machine := AsMachine(i)
			if machine.BootEnv != b.Name {
				continue
			}
			e.Errorf("Bootenv %s in use by Machine %s", b.Name, machine.Name)
		}
	}
	return e.HasError()
}

func (b *BootEnv) AfterDelete() {
	if b.OnlyUnknown {
		err := &models.Error{Object: b}
		rts := b.Render(b.stores, nil, err)
		if err.ContainsError() {
			b.Errors = err.Messages
		} else {
			rts.deregister(b.p.FS)
		}
	}
}

func AsBootEnv(o models.Model) *BootEnv {
	return o.(*BootEnv)
}

func AsBootEnvs(o []models.Model) []*BootEnv {
	res := make([]*BootEnv, len(o))
	for i := range o {
		res[i] = AsBootEnv(o[i])
	}
	return res
}

func (b *BootEnv) renderInfo() ([]models.TemplateInfo, []string) {
	return b.Templates, b.RequiredParams
}

func (b *BootEnv) templates() *template.Template {
	return b.rootTemplate
}

func (b *BootEnv) Render(d Stores, m *Machine, e models.ErrorAdder) renderers {
	if len(b.RequiredParams) > 0 && m == nil {
		e.Errorf("Machine is nil or does not have params")
		return nil
	}
	r := newRenderData(d, b.p, m, b)
	return r.makeRenderers(e)
}

func (b *BootEnv) AfterSave() {
	if b.Available && b.renderers != nil {
		b.renderers.register(b.p.FS)
	}
	b.renderers = nil
}

var bootEnvLockMap = map[string][]string{
	"get":    []string{"bootenvs"},
	"create": []string{"bootenvs", "machines", "tasks", "templates", "profiles"},
	"update": []string{"bootenvs", "machines", "tasks", "templates", "profiles"},
	"patch":  []string{"bootenvs", "machines", "tasks", "templates", "profiles"},
	"delete": []string{"bootenvs", "machines", "tasks", "templates", "profiles"},
}

func (b *BootEnv) Locks(action string) []string {
	return bootEnvLockMap[action]
}
