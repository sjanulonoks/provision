package backend

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Template represents a template that will be associated with a boot
// environment.
//
// swagger:model
type Template struct {
	*models.Template
	validate
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
			func(i, j models.Model) bool { return fix(i).ID < fix(j).ID },
			func(ref models.Model) (gte, gt index.Test) {
				refID := fix(ref).ID
				return func(s models.Model) bool {
						return fix(s).ID >= refID
					},
					func(s models.Model) bool {
						return fix(s).ID > refID
					}
			},
			func(s string) (models.Model, error) {
				tmpl := &Template{}
				tmpl.ID = s
				return tmpl, nil
			}),
	}
}

func (t *Template) Backend() store.Store {
	return t.p.getBackend(t)
}

func (t *Template) AuthKey() string {
	return t.Key()
}

func (t *Template) New() store.KeySaver {
	return &Template{Template: &models.Template{}}
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

func (t *Template) checkSubs(root *template.Template, e models.ErrorAdder) {
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

func (t *Template) Validate() {
	if t.ID == "" {
		t.Errorf("Template must have an ID")
		return
	}
	t.p.tmplMux.Lock()
	root, err := t.p.rootTemplate.Clone()
	t.p.tmplMux.Unlock()
	if err != nil {
		t.Errorf("Error cloning shared template namespace: %v", err)
		return
	}
	if err := t.parse(root); err != nil {
		t.Errorf("Parse error for template %s: %v", t.ID, err)
		return
	}
	t.AddError(index.CheckUnique(t, t.stores("templates").Items()))
	if t.HasError() != nil {
		return
	}
	t.checkSubs(root, t)
	t.SetValid()
	t.SetAvailable()
}

func (t *Template) BeforeSave() error {
	t.Validate()
	if !t.Useable() {
		return t.MakeError(422, ValidationError, t)
	}
	return nil
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

func (t *Template) OnLoad() error {
	return t.BeforeSave()
}

func (t *Template) BeforeDelete() error {
	e := &models.Error{Code: 409, Type: StillInUseError, Object: t}
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

func AsTemplate(o models.Model) *Template {
	return o.(*Template)
}

func AsTemplates(o []models.Model) []*Template {
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
