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
	"github.com/digitalrebar/store"
)

// OsInfo holds information about the operating system this BootEnv
// maps to.  Most of this information is optional for now.
//
// swagger:model
type OsInfo struct {
	// The name of the OS this BootEnv has.
	//
	// required: true
	Name string
	// The family of operating system (linux distro lineage, etc)
	Family string
	// The codename of the OS, if any.
	Codename string
	// The version of the OS, if any.
	Version string
	// The name of the ISO that the OS should install from.
	IsoFile string
	// The SHA256 of the ISO file.  Used to check for corrupt downloads.
	IsoSha256 string
	// The URL that the ISO can be downloaded from, if any.
	//
	// swagger:strfmt uri
	IsoUrl string
}

// BootEnv encapsulates the machine-agnostic information needed by the
// provisioner to set up a boot environment.
//
// swagger:model
type BootEnv struct {
	Validation
	validate
	// The name of the boot environment.  Boot environments that install
	// an operating system must end in '-install'.
	//
	// required: true
	Name string
	// A description of this boot environment.  This should tell what
	// the boot environment is for, any special considerations that
	// shoudl be taken into account when using it, etc.
	Description string
	// The OS specific information for the boot environment.
	OS OsInfo
	// The templates that should be expanded into files for the
	// boot environment.
	//
	// required: true
	Templates []TemplateInfo
	// The partial path to the kernel for the boot environment.  This
	// should be path that the kernel is located at in the OS ISO or
	// install archive.
	//
	// required: true
	Kernel string
	// Partial paths to the initrds that should be loaded for the boot
	// environment. These should be paths that the initrds are located
	// at in the OS ISO or install archive.
	//
	// required: true
	Initrds []string
	// A template that will be expanded to create the full list of
	// boot parameters for the environment.
	//
	// required: true
	BootParams string
	// The list of extra required parameters for this
	// bootstate. They should be present as Machine.Params when
	// the bootenv is applied to the machine.
	//
	// required: true
	RequiredParams []string
	// The list of extra optional parameters for this
	// bootstate. They can be present as Machine.Params when
	// the bootenv is applied to the machine.  These are more
	// other consumers of the bootenv to know what parameters
	// could additionally be applied to the bootenv by the
	// renderer based upon the Machine.Params
	//
	OptionalParams []string
	// OnlyUnknown indicates whether this bootenv can be used without a
	// machine.  Only bootenvs with this flag set to `true` be used for
	// the unknownBootEnv preference.
	//
	// required: true
	OnlyUnknown bool
	// The list of initial machine tasks that the boot environment should get
	Tasks          []string
	bootParamsTmpl *template.Template
	p              *DataTracker
	rootTemplate   *template.Template
	tmplMux        sync.Mutex
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
			func(i, j store.KeySaver) bool {
				return fix(i).Name < fix(j).Name
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				name := fix(ref).Name
				return func(s store.KeySaver) bool {
						return fix(s).Name >= name
					},
					func(s store.KeySaver) bool {
						return fix(s).Name > name
					}
			},
			func(s string) (store.KeySaver, error) {
				return &BootEnv{Name: s}, nil
			}),
		"Available": index.Make(
			false,
			"boolean",
			func(i, j store.KeySaver) bool {
				return (!fix(i).Available) && fix(j).Available
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				avail := fix(ref).Available
				return func(s store.KeySaver) bool {
						v := fix(s).Available
						return v || (v == avail)
					},
					func(s store.KeySaver) bool {
						return fix(s).Available && !avail
					}
			},
			func(s string) (store.KeySaver, error) {
				res := &BootEnv{}
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
			func(i, j store.KeySaver) bool {
				return !fix(i).OnlyUnknown && fix(j).OnlyUnknown
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				unknown := fix(ref).OnlyUnknown
				return func(s store.KeySaver) bool {
						v := fix(s).OnlyUnknown
						return v || (v && unknown)
					},
					func(s store.KeySaver) bool {
						return fix(s).OnlyUnknown && !unknown
					}
			},
			func(s string) (store.KeySaver, error) {
				res := &BootEnv{}
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

func (b *BootEnv) genRoot(commonRoot *template.Template, e *Error) *template.Template {
	res := MergeTemplates(commonRoot, b.Templates, e)
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
	if e.containsError {
		return nil
	}
	return res
}

func (b *BootEnv) OnLoad() error {
	e := &Error{o: b}
	b.tmplMux.Lock()
	defer b.tmplMux.Unlock()
	b.p.tmplMux.Lock()
	defer b.p.tmplMux.Unlock()
	b.rootTemplate = b.genRoot(b.p.rootTemplate, e)
	b.Errors = e.Messages
	b.Available = !e.containsError
	return nil
}

func (b *BootEnv) Prefix() string {
	return "bootenvs"
}

func (b *BootEnv) Key() string {
	return b.Name
}

func (b *BootEnv) AuthKey() string {
	return b.Key()
}

func (b *BootEnv) New() store.KeySaver {
	return &BootEnv{Name: b.Name, p: b.p}
}

func (b *BootEnv) setDT(p *DataTracker) {
	b.p = p
}

func (b *BootEnv) explodeIso(e *Error) {
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
		e.Errorf("Explode ISO: iso doesn't exist: %s\n", isoPath)
		if b.OS.IsoUrl != "" {
			e.Errorf("You can download the required ISO from %s", b.OS.IsoUrl)
		}
		return
	}

	// Only check the has if we have one.
	if b.OS.IsoSha256 != "" {
		f, err := os.Open(isoPath)
		if err != nil {
			e.Errorf("Explode ISO: failed to open iso file %s: %v", isoPath, err)
			return
		}
		defer f.Close()
		hasher := sha256.New()
		if _, err := io.Copy(hasher, f); err != nil {
			e.Errorf("Explode ISO: failed to read iso file %s: %v", isoPath, err)
			return
		}
		hash := hex.EncodeToString(hasher.Sum(nil))
		if hash != b.OS.IsoSha256 {
			e.Errorf("Explode ISO: SHA256 bad. actual: %v expected: %v", hash, b.OS.IsoSha256)
			return
		}
	}

	// Call extract script
	// /explode_iso.sh b.OS.Name fileRoot isoPath path.Dir(canaryPath)
	cmdName := path.Join(b.p.FileRoot, "explode_iso.sh")
	cmdArgs := []string{b.OS.Name, b.p.FileRoot, isoPath, b.localPathFor(""), b.OS.IsoSha256}
	if out, err := exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		e.Errorf("Explode ISO: explode_iso.sh failed for %s: %s", b.Name, err)
		e.Errorf("Command output:\n%s", string(out))

	} else {
		b.p.Infof("debugBootEnv", "Explode ISO: %s exploded to %s", b.OS.IsoFile, isoPath)
		b.p.Debugf("debugBootEnv", "Explode ISO Log:\n%s", string(out))
	}
	return
}

func (b *BootEnv) Validate() error {
	e := &Error{Code: 422, Type: ValidationError, o: b}
	if err := index.CheckUnique(b, b.stores("bootenvs").Items()); err != nil {
		e.Merge(err)
	}
	for _, taskName := range b.Tasks {
		if b.stores("tasks").Find(taskName) == nil {
			e.Errorf("Task %s does not exist", taskName)
		}
	}
	// If our basic templates do not parse, it is game over for us
	b.p.tmplMux.Lock()
	b.tmplMux.Lock()
	root := b.genRoot(b.p.rootTemplate, e)
	b.p.tmplMux.Unlock()
	if root != nil {
		b.rootTemplate = root
	}
	b.tmplMux.Unlock()
	return e.OrNil()
}

func (b *BootEnv) BeforeSave() error {
	if err := b.Validate(); err != nil {
		return err
	}

	e := &Error{Code: 422, Type: ValidationError, o: b}

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
			e.Errorf("bootenv: Missing elilo or pxelinux template")
		}
	}
	// Make sure the ISO for this bootenv has been exploded locally so that
	// the boot env can use its contents.
	if b.OS.IsoFile != "" {
		b.explodeIso(e)
	}
	// If we have a non-empty Kernel, make sure it points at something kernel-ish.
	if b.Kernel != "" {
		kPath := b.localPathFor(b.Kernel)
		kernelStat, err := os.Stat(kPath)
		if err != nil {
			e.Errorf("bootenv: %s: missing kernel %s (%s)",
				b.Name,
				b.Kernel,
				kPath)
		} else if !kernelStat.Mode().IsRegular() {
			e.Errorf("bootenv: %s: invalid kernel %s (%s)",
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
				e.Errorf("bootenv: %s: missing initrd %s (%s)",
					b.Name,
					initrd,
					iPath)
				continue
			}
			if !initrdStat.Mode().IsRegular() {
				e.Errorf("bootenv: %s: invalid initrd %s (%s)",
					b.Name,
					initrd,
					iPath)
			}
		}
	}
	b.Errors = e.Messages
	b.Available = !e.ContainsError()
	b.Validated = true
	return nil
}

func (b *BootEnv) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: b}
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
	return e.OrNil()
}

func (b *BootEnv) AfterDelete() {
	if b.OnlyUnknown {
		err := &Error{o: b}
		rts := b.Render(b.stores, nil, err)
		if err.ContainsError() {
			b.Errors = err.Messages
		} else {
			rts.deregister(b.p.FS)
		}
	}
}

func (p *DataTracker) NewBootEnv() *BootEnv {
	return &BootEnv{p: p}
}

func AsBootEnv(o store.KeySaver) *BootEnv {
	return o.(*BootEnv)
}

func AsBootEnvs(o []store.KeySaver) []*BootEnv {
	res := make([]*BootEnv, len(o))
	for i := range o {
		res[i] = AsBootEnv(o[i])
	}
	return res
}

func (b *BootEnv) renderInfo() ([]TemplateInfo, []string) {
	return b.Templates, b.RequiredParams
}

func (b *BootEnv) templates() *template.Template {
	return b.rootTemplate
}

func (b *BootEnv) Render(d Stores, m *Machine, e *Error) renderers {
	if len(b.RequiredParams) > 0 && m == nil {
		e.Errorf("Machine is nil or does not have params")
		return nil
	}
	r := newRenderData(d, b.p, m, b)
	return r.makeRenderers(e)
}

func (b *BootEnv) AfterSave() {
	if b.OnlyUnknown {
		err := &Error{o: b}
		rts := b.Render(b.stores, nil, err)
		if err.ContainsError() {
			b.Errors = err.Messages
		} else {
			rts.register(b.p.FS)
		}
		return
	}
	machines := b.stores("machines")
	for _, i := range machines.Items() {
		machine := AsMachine(i)
		if machine.BootEnv != b.Name {
			continue
		}
		err := &Error{o: b}
		rts := b.Render(b.stores, machine, err)
		if err.ContainsError() {
			machine.Errors = err.Messages
		} else {
			rts.register(b.p.FS)
		}
	}
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
