package backend

import (
	"sync"
	"text/template"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Task is a thing that can run on a Machine.
//
// swagger:model
type Task struct {
	*models.Task
	validate
	p            *DataTracker
	rootTemplate *template.Template
	tmplMux      sync.Mutex
}

func (obj *Task) SaveClean() store.KeySaver {
	mod := *obj.Task
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func AsTask(o models.Model) *Task {
	return o.(*Task)
}

func AsTasks(o []models.Model) []*Task {
	res := make([]*Task, len(o))
	for i := range o {
		res[i] = AsTask(o[i])
	}
	return res
}

func (t *Task) Backend() store.Store {
	return t.p.getBackend(t)
}

func (t *Task) New() store.KeySaver {
	res := &Task{Task: &models.Task{}}
	res.p = t.p
	return res
}

func (t *Task) setDT(dp *DataTracker) {
	t.p = dp
}

func (t *Task) Indexes() map[string]index.Maker {
	fix := AsTask
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Name": index.Make(
			true,
			"string",
			func(i, j models.Model) bool { return fix(i).Name < fix(j).Name },
			func(ref models.Model) (gte, gt index.Test) {
				refName := fix(ref).Name
				return func(s models.Model) bool {
						return fix(s).Name >= refName
					},
					func(s models.Model) bool {
						return fix(s).Name > refName
					}
			},
			func(s string) (models.Model, error) {
				task := fix(t.New())
				task.Name = s
				return task, nil
			}),
	}
}

func (t *Task) genRoot(common *template.Template, e models.ErrorAdder) *template.Template {
	res := models.MergeTemplates(common, t.Templates, e)
	if e.HasError() != nil {
		return nil
	}
	return res
}

func (t *Task) Validate() {
	t.tmplMux.Lock()
	defer t.tmplMux.Unlock()
	t.p.tmplMux.Lock()
	defer t.p.tmplMux.Unlock()
	root := t.genRoot(t.p.rootTemplate, t)
	t.SetValid()
	if t.Useable() {
		t.rootTemplate = root
		t.SetAvailable()
	}
	return
}

func (t *Task) OnLoad() error {
	stores, unlocker := t.p.LockAll()
	t.stores = stores
	defer func() { unlocker(); t.stores = nil }()
	t.Validate()
	if !t.Useable() {
		return t.MakeError(422, ValidationError, t)
	}
	return nil
}

func (t *Task) BeforeSave() error {
	return t.OnLoad()
}

type taskHaver interface {
	models.Model
	HasTask(string) bool
}

func (t *Task) BeforeDelete() error {
	e := &models.Error{Code: 409, Type: StillInUseError, Object: t}
	for _, objPrefix := range []string{"profiles", "machines", "bootenvs"} {
		for _, j := range t.stores(objPrefix).Items() {
			thing := j.(taskHaver)
			if thing.HasTask(t.Name) {
				e.Errorf("%s:%s still uses %s", thing.Prefix(), thing.Key(), t.Name)
			}
		}
	}
	return e.HasError()
}

func (t *Task) renderInfo() ([]models.TemplateInfo, []string) {
	return t.Templates, t.RequiredParams
}

func (t *Task) templates() *template.Template {
	return t.rootTemplate
}

func (t *Task) Render(d Stores, m *Machine, e *models.Error) renderers {
	if m == nil {
		e.Errorf("No machine to render against")
		return nil
	}
	r := newRenderData(d, t.p, m, t)
	return r.makeRenderers(e)
}

var taskLockMap = map[string][]string{
	"get":    []string{"templates", "tasks"},
	"create": []string{"templates", "tasks"},
	"update": []string{"templates", "tasks"},
	"patch":  []string{"templates", "tasks"},
	"delete": []string{"bootenvs", "tasks", "profiles", "machines"},
}

func (t *Task) Locks(action string) []string {
	return taskLockMap[action]
}
