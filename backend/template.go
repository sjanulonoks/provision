package backend

import (
	"fmt"
	"io"
	"text/template"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

// Template represents a template that will be associated with a boot
// environment.
//
// swagger:model
type Template struct {
	// ID is a unique identifier for this template.
	//
	// required: true
	ID string
	// A description of this template
	Description string
	// Contents is the raw template.  It must be a valid template
	// according to text/template.
	//
	// required: true
	Contents   string
	parsedTmpl *template.Template
	p          *DataTracker
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

// Parse checks to make sure the template contents are valid according to text/template.
func (t *Template) parse() error {
	e := &Error{Code: 422, Type: ValidationError, o: t}
	parsedTmpl, err := template.New(t.ID).Parse(t.Contents)
	if err != nil {
		e.Errorf("%v", err)
		return e
	}
	t.parsedTmpl = parsedTmpl.Option("missingkey=error")
	return nil
}

func (t *Template) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: t}
	if t.ID == "" {
		e.Errorf("Template must have an ID")
	}
	if err := t.parse(); err != nil {
		e.Errorf("Parse error: %v", err)
	}
	return e.OrNil()
}

func (t *Template) BootEnvs() []*BootEnv {
	return AsBootEnvs(t.p.FetchAll(t.p.NewBootEnv()))
}

func (t *Template) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: t}
	for _, bootEnv := range t.BootEnvs() {
		for _, tmpl := range bootEnv.Templates {
			if tmpl.ID == t.ID {
				e.Errorf("In use by bootenv %s (as %s)", bootEnv.Name, tmpl.Name)
			}
		}
	}
	return e.OrNil()
}

// Render executes the template with params writing the results to dest
func (t *Template) render(dest io.Writer, params interface{}) error {
	if t.parsedTmpl == nil {
		if err := t.parse(); err != nil {
			return fmt.Errorf("template: %s does not compile: %v", t.ID, err)
		}
	}
	if err := t.parsedTmpl.Execute(dest, params); err != nil {
		return fmt.Errorf("template: cannot execute %s: %v", t.ID, err)
	}
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
