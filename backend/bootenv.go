package backend

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/digitalrebar/go/rebar-api/api"
)

// TemplateInfo holds information on the templates in the boot
// environment that will be expanded into files.
//
// swagger:model
type TemplateInfo struct {
	// Name of the template
	//
	// required: true
	Name string
	// A text/template that specifies how to create
	// the final path the template should be
	// written to.
	//
	// required: true
	Path string
	// The ID of the template that should be expanded.  Either
	// this or Contents should be set
	//
	// required: false
	ID string
	// The contents that should be used when this template needs
	// to be expanded.  Either this or ID should be set.
	//
	// required: false
	Contents string
	pathTmpl *template.Template
}

func (t *TemplateInfo) contents(dt *DataTracker) (*Template, bool) {
	if t.ID != "" {
		res, found := dt.fetchOne(dt.NewTemplate(), t.ID)
		if found {
			return AsTemplate(res), found
		}
		return nil, found
	} else {
		return &Template{ID: t.Name, Contents: t.Contents}, true
	}
}

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
	// Whether the boot environment is useable.  This can only be set to
	// `true` if there are no Errors, and if there are any errors
	// Avaialble will be set to `false`, and will require user
	// intervention to set it back to `true`.
	//
	// required: true
	Available bool
	// Any errors that were recorded in the processing of this boot
	// environment.
	//
	// read only: true
	Errors         []string
	bootParamsTmpl *template.Template
	p              *DataTracker
}

func (b *BootEnv) Backend() store.SimpleStore {
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

func (b *BootEnv) parseTemplates(e *Error) {
	for i := range b.Templates {
		ti := &b.Templates[i]
		if ti.Name == "" {
			e.Errorf("Templates[%d] has no Name", i)
		}
		if ti.Path == "" {
			e.Errorf("Templates[%d] has no Path", i)
		} else {
			pathTmpl, err := template.New(ti.Name).Parse(ti.Path)
			if err != nil {
				e.Errorf("Error compiling path template %s (%s): %v",
					ti.Name,
					ti.Path,
					err)
				continue
			} else {
				ti.pathTmpl = pathTmpl.Option("missingkey=error")
			}
		}
		if ti.ID != "" {
			tmpl := b.p.NewTemplate()
			if _, found := b.p.fetchOne(tmpl, ti.ID); !found {
				e.Errorf("Templates[%d] wants Template %s, which does not exist",
					i,
					ti.ID)
			}
		} else if ti.Contents == "" {
			e.Errorf("Templates[%d] has neither ID nor Contents, one of the other must be non-empty.", i)
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
	return
}

func (b *BootEnv) OnLoad() error {
	e := &Error{o: b}
	b.parseTemplates(e)
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

func (b *BootEnv) New() store.KeySaver {
	return &BootEnv{Name: b.Name, p: b.p}
}

func (b *BootEnv) setDT(p *DataTracker) {
	b.p = p
}

func (b *BootEnv) explodeIso(e *Error) {
	// Only work on things that are requested.
	if b.OS.IsoFile == "" {
		b.p.Logger.Printf("Explode ISO: Skipping %s becausing no iso image specified\n", b.Name)
		return
	}
	// Have we already exploded this?  If file exists, then good!
	canaryPath := b.localPathFor("." + b.OS.Name + ".rebar_canary")
	buf, err := ioutil.ReadFile(canaryPath)
	if err == nil && len(buf) != 0 && string(bytes.TrimSpace(buf)) == b.OS.IsoSha256 {
		b.p.Logger.Printf("Explode ISO: canary file %s, in place and has proper SHA256\n", canaryPath)
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
	// This will wind up being saved along with the rest of the
	// hash because explodeIso is called by OnChange before the struct gets saved.
	if b.OS.IsoSha256 == "" {
		b.OS.IsoSha256 = hash
	}

	if hash != b.OS.IsoSha256 {
		e.Errorf("Explode ISO: SHA256 bad. actual: %v expected: %v", hash, b.OS.IsoSha256)
		return
	}

	// Call extract script
	// /explode_iso.sh b.OS.Name fileRoot isoPath path.Dir(canaryPath)
	cmdName := path.Join(b.p.FileRoot, "explode_iso.sh")
	cmdArgs := []string{b.OS.Name, b.p.FileRoot, isoPath, b.localPathFor(""), b.OS.IsoSha256}
	if out, err := exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		e.Errorf("Explode ISO: explode_iso.sh failed for %s: %s", b.Name, err)
		e.Errorf("Command output:\n%s", string(out))

	} else {
		b.p.Logger.Printf("Explode ISO: %s exploded to %s", b.OS.IsoFile, isoPath)
		b.p.Logger.Printf("Explode ISO Log:\n%s", string(out))
	}
	return
}

func (b *BootEnv) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: b}
	// If our basic templates do not parse, it is game over for us
	b.parseTemplates(e)
	if e.containsError {
		return e
	}
	// Otherwise, we will save the BootEnv, but record
	// the list of errors and mark it as not available.
	//
	// First, we have to have an iPXE template, or a PXELinux and eLILO template, or all three.
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
	b.Available = (len(b.Errors) == 0)

	return nil
}

func (b *BootEnv) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: b}
	machines := AsMachines(b.p.FetchAll(b.p.NewMachine()))
	for _, machine := range machines {
		if machine.BootEnv != b.Name {
			continue
		}
		e.Errorf("Bootenv %s in use by Machine %s", b.Name, machine.Name)
	}
	return e.OrNil()
}

func (b *BootEnv) List() []*BootEnv {
	return AsBootEnvs(b.p.FetchAll(b))
}

func (b *BootEnv) AfterSave() {
	b.rebuildRebarData()
}

func (b *BootEnv) AfterDelete() {
	b.rebuildRebarData()
}

func (b *BootEnv) rebuildRebarData() {
	var err error
	if b.p.RebarClient == nil {
		return
	}
	preferredOses := map[string]int{
		"centos-7.3.1611": 0,
		"centos-7.2.1511": 1,
		"centos-7.1.1503": 2,
		"ubuntu-16.04":    3,
		"ubuntu-14.04":    4,
		"ubuntu-15.04":    5,
		"debian-8":        6,
		"centos-6.8":      7,
		"centos-6.6":      8,
		"debian-7":        9,
		"redhat-6.5":      10,
		"ubuntu-12.04":    11,
	}

	attrValOSes := make(map[string]bool)
	attrValOS := "STRING"
	attrPref := 1000

	if !b.Available {
		return
	}

	bes := b.List()

	if bes == nil || len(bes) == 0 {
		b.p.Logger.Printf("No boot environments, nothing to do")
		return
	}

	for _, be := range bes {
		if !strings.HasSuffix(be.Name, "-install") {
			continue
		}
		if !be.Available {
			continue
		}
		attrValOSes[be.OS.Name] = true
		numPref, ok := preferredOses[be.OS.Name]
		if !ok {
			numPref = 999
		}
		if numPref < attrPref {
			attrValOS = be.OS.Name
			attrPref = numPref
		}
	}

	deployment := &api.Deployment{}
	if err := b.p.RebarClient.Fetch(deployment, "system"); err != nil {
		b.p.Logger.Printf("Failed to load system deployment: %v", err)
		return
	}

	role := &api.Role{}
	if err := b.p.RebarClient.Fetch(role, "provisioner-service"); err != nil {
		b.p.Logger.Printf("Failed to fetch provisioner-service: %v", err)
		return
	}

	var tgt api.Attriber
	for {
		drs := []*api.DeploymentRole{}
		matcher := make(map[string]interface{})
		matcher["role_id"] = role.ID
		matcher["deployment_id"] = deployment.ID
		dr := &api.DeploymentRole{}
		if err := b.p.RebarClient.Match(b.p.RebarClient.UrlPath(dr), matcher, &drs); err != nil {
			b.p.Logger.Printf("Failed to find deployment role to update: %v", err)
			return
		}
		if len(drs) != 0 {
			tgt = drs[0]
			break
		}
		b.p.Logger.Printf("Waiting for provisioner-service (%v) to show up in system(%v)", role.ID, deployment.ID)
		b.p.Logger.Printf("drs: %#v, err: %#v", drs, err)
		time.Sleep(5 * time.Second)
	}

	attrib := &api.Attrib{}
	attrib.SetId("provisioner-available-oses")
	attrib, err = b.p.RebarClient.GetAttrib(tgt, attrib, "")
	if err != nil {
		b.p.Logger.Printf("Failed to fetch provisioner-available-oses: %v", err)
		return
	}
	attrib.Value = attrValOSes
	if err := b.p.RebarClient.SetAttrib(tgt, attrib, ""); err != nil {
		b.p.Logger.Printf("Failed to update provisioner-available-oses: %v", err)
		return
	}

	attrib = &api.Attrib{}
	attrib.SetId("provisioner-default-os")
	attrib, err = b.p.RebarClient.GetAttrib(tgt, attrib, "")
	if err != nil {
		b.p.Logger.Printf("Failed to get default OS: %v:", err)
		return
	}
	attrib.Value = attrValOS
	if err := b.p.RebarClient.SetAttrib(tgt, attrib, ""); err != nil {
		b.p.Logger.Printf("Failed to set default OS: %v", err)
		return
	}

	if err := b.p.RebarClient.Commit(tgt); err != nil {
		b.p.Logger.Printf("Failed to commit changes: %v", err)
		return
	}

	return
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
