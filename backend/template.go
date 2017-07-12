package backend

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
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

func (ti *TemplateInfo) id() string {
	if ti.ID == "" {
		return ti.Name
	}
	return ti.ID
}

func MergeTemplates(root *template.Template, tmpls []TemplateInfo, e *Error) *template.Template {
	var res *template.Template
	var err error
	if root == nil {
		res = template.New("")
	} else {
		res, err = root.Clone()
	}
	if err != nil {
		e.Errorf("Error cloning root: %v", err)
		return nil
	}
	buf := &bytes.Buffer{}
	for i := range tmpls {
		ti := &tmpls[i]
		if ti.Name == "" {
			e.Errorf("Templates[%d] has no Name", i)
			continue
		}
		if ti.Path != "" {
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
			if res.Lookup(ti.ID) == nil {
				e.Errorf("Templates[%d]: No common template for %s", i, ti.ID)
			}
			continue
		}
		if ti.Contents == "" {
			e.Errorf("Templates[%d] has both an empty ID and contents", i)
		}
		fmt.Fprintf(buf, `{{define "%s"}}%s{{end}}\n`, ti.Name, ti.Contents)
	}
	_, err = res.Parse(buf.String())
	if err != nil {
		e.Errorf("Error parsing inline templates: %v", err)
		return nil
	}
	if e.containsError {
		return nil
	}
	return res
}

// Template represents a template that will be associated with a boot
// environment.
//
// swagger:model
type Template struct {
	validate
	// ID is a unique identifier for this template.  It cannot change once it is set.
	//
	// required: true
	ID string
	// A description of this template
	Description string
	// Contents is the raw template.  It must be a valid template
	// according to text/template.
	//
	// required: true
	Contents string
	p        *DataTracker
	toUpdate *tmplUpdater
}

func (p *Template) Indexes() map[string]index.Maker {
	fix := AsTemplate
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"ID": index.Make(
			true,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).ID < fix(j).ID },
			func(ref store.KeySaver) (gte, gt index.Test) {
				refID := fix(ref).ID
				return func(s store.KeySaver) bool {
						return fix(s).ID >= refID
					},
					func(s store.KeySaver) bool {
						return fix(s).ID > refID
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Template{ID: s}, nil
			}),
	}
}

func (t *Template) Prefix() string {
	return "templates"
}

func (t *Template) Backend() store.SimpleStore {
	return t.p.getBackend(t)
}

func (t *Template) Key() string {
	return t.ID
}

func (t *Template) New() store.KeySaver {
	res := &Template{ID: t.ID, p: t.p}
	return store.KeySaver(res)
}

func (t *Template) setDT(p *DataTracker) {
	t.p = p
}

func (t *Template) parse(root *template.Template) error {
	_, err := root.New(t.ID).Parse(t.Contents)
	return err
}

type tmplUpdater struct {
	root                *template.Template
	tasks               []*Task
	bootenvs            []*BootEnv
	taskTmpls, envTmpls []*template.Template
}

func (t *Template) checkSubs(root *template.Template, e *Error) {
	t.toUpdate = &tmplUpdater{
		root:     root,
		tasks:    AsTasks(t.stores("tasks").Items()),
		bootenvs: AsBootEnvs(t.stores("bootenvs").Items()),
	}
	t.toUpdate.taskTmpls = make([]*template.Template, len(t.toUpdate.tasks))
	t.toUpdate.envTmpls = make([]*template.Template, len(t.toUpdate.bootenvs))
	for i, task := range t.toUpdate.tasks {
		t.toUpdate.taskTmpls[i] = task.genRoot(root, e)
	}
	for i, bootenv := range t.toUpdate.bootenvs {
		t.toUpdate.envTmpls[i] = bootenv.genRoot(root, e)
	}
}

func (t *Template) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: t}
	if t.ID == "" {
		e.Errorf("Template must have an ID")
		return e
	}
	t.p.tmplMux.Lock()
	root, err := t.p.rootTemplate.Clone()
	t.p.tmplMux.Unlock()
	if err != nil {
		e.Errorf("Error cloning shared template namespace: %v", err)
		return e
	}
	if err := t.parse(root); err != nil {
		e.Errorf("Parse error for template %s: %v", t.ID, err)
		return e
	}
	e.Merge(index.CheckUnique(t, t.stores("templates").Items()))
	if e.ContainsError() {
		return e
	}
	t.checkSubs(root, e)
	return e.OrNil()
}

func (t *Template) updateOthers() {
	t.p.tmplMux.Lock()
	t.p.rootTemplate = t.toUpdate.root
	t.p.tmplMux.Unlock()
	for i, task := range t.toUpdate.tasks {
		task.tmplMux.Lock()
		task.rootTemplate = t.toUpdate.taskTmpls[i]
		task.tmplMux.Unlock()
	}
	for i, bootenv := range t.toUpdate.bootenvs {
		bootenv.tmplMux.Lock()
		bootenv.rootTemplate = t.toUpdate.envTmpls[i]
		bootenv.tmplMux.Unlock()
	}
	t.toUpdate = nil
}

func (t *Template) AfterSave() {
	t.updateOthers()
}

func (t *Template) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: t}
	buf := &bytes.Buffer{}
	for _, i := range t.stores("templates").Items() {
		tmpl := AsTemplate(i)
		if tmpl.ID == t.ID {
			continue
		}
		fmt.Fprintf(buf, `{{define "%s"}}%s{{end}}\n`, tmpl.ID, tmpl.Contents)
	}
	root, err := template.New("").Parse(buf.String())
	if err != nil {
		e.Errorf("Template %s still required: %v", t.ID, err)
		return e
	}
	t.checkSubs(root, e)
	if e.ContainsError() {
		return e
	}
	t.updateOthers()
	return nil
}

func (p *DataTracker) NewTemplate() *Template {
	return &Template{p: p}
}

func AsTemplate(o store.KeySaver) *Template {
	return o.(*Template)
}

func AsTemplates(o []store.KeySaver) []*Template {
	res := make([]*Template, len(o))
	for i := range o {
		res[i] = AsTemplate(o[i])
	}
	return res
}

var templateLockMap = map[string][]string{
	"get":    []string{"templates"},
	"create": []string{"templates", "bootenvs", "machines", "tasks"},
	"update": []string{"templates", "bootenvs", "machines", "tasks"},
	"patch":  []string{"templates", "bootenvs", "machines", "tasks"},
	"delete": []string{"templates", "bootenvs", "machines", "tasks"},
}

func (t *Template) Locks(action string) []string {
	return templateLockMap[action]
}
