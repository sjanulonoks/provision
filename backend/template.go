package backend

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

// Template represents a template that will be associated with a boot
// environment.
//
// swagger:model
type Template struct {
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
}

func (p *Template) Indexes() map[string]index.Maker {
	fix := AsTemplate
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"ID": index.Make(
			true,
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

func (t *Template) List() []*Template {
	return AsTemplates(t.p.FetchAll(t))
}

func (t *Template) parse(root *template.Template) error {
	_, err := root.New(t.ID).Parse(t.Contents)
	return err
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
	if err := index.CheckUnique(t, t.p.objs[t.Prefix()].d); err != nil {
		e.Merge(err)
		return e
	}
	bootEnvs := t.p.lockFor("bootenvs")
	defer bootEnvs.Unlock()
	for _, env := range bootEnvs.d {
		AsBootEnv(env).genRoot(root, e)
	}
	return e.OrNil()
}

func (t *Template) AfterSave() {
	t.p.tmplMux.Lock()
	defer t.p.tmplMux.Unlock()
	root, err := t.p.rootTemplate.Clone()
	if err != nil {
		t.p.Printf("Error cloning shared template namespace: %v", err)
		return
	}
	if err := t.parse(root); err != nil {
		t.p.Printf("Parse error for template %s: %v", t.ID, err)
		return
	}
	bootEnvs := t.p.lockFor("bootenvs")
	defer bootEnvs.Unlock()
	newRoots := make([]*template.Template, len(bootEnvs.d))
	for i, envIsh := range bootEnvs.d {
		env := AsBootEnv(envIsh)
		env.tmplMux.Lock()
		defer env.tmplMux.Unlock()
		e := &Error{o: env}
		newRoots[i] = env.genRoot(root, e)
		if e.containsError {
			t.p.Logger.Print(e.Error())
			return
		}
	}
	t.p.rootTemplate = root
	for i, envIsh := range bootEnvs.d {
		env := AsBootEnv(envIsh)
		env.rootTemplate = newRoots[i]
	}
}

func (t *Template) BootEnvs() []*BootEnv {
	return AsBootEnvs(t.p.FetchAll(t.p.NewBootEnv()))
}

func (t *Template) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: t}
	buf := &bytes.Buffer{}
	templates := t.p.objs["templates"].d
	for i := range templates {
		tmpl := AsTemplate(templates[i])
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
	t.p.tmplMux.Lock()
	defer t.p.tmplMux.Unlock()
	bootEnvs := t.p.lockFor("bootenvs")
	benvRoots := make([]*template.Template, len(bootEnvs.d))
	defer bootEnvs.Unlock()
	for i := range bootEnvs.d {
		env := AsBootEnv(bootEnvs.d[i])
		benvRoots[i] = env.genRoot(root, e)
	}
	if e.containsError {
		return e
	}
	for i := range benvRoots {
		env := AsBootEnv(bootEnvs.d[i])
		env.tmplMux.Lock()
		env.rootTemplate = benvRoots[i]
		env.tmplMux.Unlock()
	}
	t.p.rootTemplate = root
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
