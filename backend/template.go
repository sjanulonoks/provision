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

func (obj *Template) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Template) SaveClean() store.KeySaver {
	mod := *obj.Template
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func (p *Template) Indexes() map[string]index.Maker {
	fix := AsTemplate
	res := index.MakeBaseIndexes(p)
	res["ID"] = index.Make(
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
			tmpl := fix(p.New())
			tmpl.ID = s
			return tmpl, nil
		})
	return res
}

func (t *Template) Backend() store.Store {
	return t.p.getBackend(t)
}

func (t *Template) New() store.KeySaver {
	res := &Template{Template: &models.Template{}}
	if t.Template != nil && t.ChangeForced() {
		res.ForceChange()
	}
	res.p = t.p
	return res
}

func (t *Template) setDT(p *DataTracker) {
	t.p = p
}

func (t *Template) parse(root *template.Template) error {
	_, err := root.New(t.ID).Parse(t.Contents)
	return err
}

type tmplUpdater struct {
	root                            *template.Template
	tasks                           []*Task
	bootenvs                        []*BootEnv
	stages                          []*Stage
	taskTmpls, envTmpls, stageTmpls []*template.Template
}

func (t *Template) checkSubs(root *template.Template, e models.ErrorAdder) {
	t.toUpdate = &tmplUpdater{root: root, tasks: []*Task{}, bootenvs: []*BootEnv{}}
	if foo := t.stores("tasks"); foo != nil {
		t.toUpdate.tasks = AsTasks(foo.Items())
	}
	if foo := t.stores("bootenvs"); foo != nil {
		t.toUpdate.bootenvs = AsBootEnvs(foo.Items())
	}
	if foo := t.stores("stages"); foo != nil {
		t.toUpdate.stages = AsStages(foo.Items())
	}
	t.toUpdate.taskTmpls = make([]*template.Template, len(t.toUpdate.tasks))
	t.toUpdate.envTmpls = make([]*template.Template, len(t.toUpdate.bootenvs))
	t.toUpdate.stageTmpls = make([]*template.Template, len(t.toUpdate.stages))
	for i, task := range t.toUpdate.tasks {
		t.toUpdate.taskTmpls[i] = task.genRoot(root, e)
	}
	for i, bootenv := range t.toUpdate.bootenvs {
		t.toUpdate.envTmpls[i] = bootenv.genRoot(root, e)
	}
	for i, stage := range t.toUpdate.stages {
		t.toUpdate.stageTmpls[i] = stage.genRoot(root, e)
	}
}

func (t *Template) Validate() {
	t.Template.Validate()
	var err error
	t.p.tmplMux.Lock()
	root := t.p.rootTemplate
	if root == nil {
		root = template.New("")
	} else {
		root, err = root.Clone()
	}
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

func (t *Template) OnLoad() error {
	t.Validated = true
	t.Available = true
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

func (t *Template) BeforeDelete() error {
	e := &models.Error{Code: 409, Type: StillInUseError, Model: t.Prefix(), Key: t.Key()}
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
	"create": []string{"stages", "templates", "bootenvs", "machines", "tasks"},
	"update": []string{"stages", "templates", "bootenvs", "machines", "tasks"},
	"patch":  []string{"stages", "templates", "bootenvs", "machines", "tasks"},
	"delete": []string{"stages", "templates", "bootenvs", "machines", "tasks"},
}

func (t *Template) Locks(action string) []string {
	return templateLockMap[action]
}
