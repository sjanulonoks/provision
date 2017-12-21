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
	rootTemplate *template.Template
	tmplMux      sync.Mutex
}

func (obj *Task) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Task) SaveClean() store.KeySaver {
	mod := *obj.Task
	mod.ClearValidation()
	return toBackend(&mod, obj.rt)
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

func (t *Task) New() store.KeySaver {
	res := &Task{Task: &models.Task{}}
	if t.Task != nil && t.ChangeForced() {
		res.ForceChange()
	}
	res.rt = t.rt
	return res
}

func (t *Task) Indexes() map[string]index.Maker {
	fix := AsTask
	res := index.MakeBaseIndexes(t)
	res["Name"] = index.Make(
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
		})
	return res
}

func (t *Task) genRoot(common *template.Template, e models.ErrorAdder) *template.Template {
	res := models.MergeTemplates(common, t.Templates, e)
	if e.HasError() != nil {
		return nil
	}
	return res
}

func (t *Task) Validate() {
	t.Task.Validate()
	t.tmplMux.Lock()
	defer t.tmplMux.Unlock()
	t.rt.dt.tmplMux.Lock()
	defer t.rt.dt.tmplMux.Unlock()
	root := t.genRoot(t.rt.dt.rootTemplate, t)
	t.SetValid()
	if t.Useable() {
		t.rootTemplate = root
		t.SetAvailable()
	}
	stages := t.rt.stores("stages")
	if stages != nil {
		for _, i := range stages.Items() {
			stage := AsStage(i)
			if stage.Tasks == nil || len(stage.Tasks) == 0 {
				continue
			}
			for _, taskName := range stage.Tasks {
				if taskName != t.Name {
					continue
				}
				func() {
					stage.rt = t.rt
					defer func() { stage.rt = nil }()
					stage.Validate()
				}()
				break
			}
		}
	}
	return
}

func (t *Task) OnLoad() error {
	defer func() { t.rt = nil }()
	return t.BeforeSave()
}

func (t *Task) BeforeSave() error {
	t.Validate()
	if !t.HasFeature("sane-exit-codes") {
		t.AddFeature("original-exit-codes")
	}
	if !t.Useable() {
		return t.MakeError(422, ValidationError, t)
	}
	return nil
}

type taskHaver interface {
	models.Model
	HasTask(string) bool
}

func (t *Task) BeforeDelete() error {
	e := &models.Error{Code: 409, Type: StillInUseError, Model: t.Prefix(), Key: t.Key()}
	for _, objPrefix := range []string{"machines", "stages"} {
		for _, j := range t.rt.stores(objPrefix).Items() {
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

func (t *Task) Render(rt *RequestTracker, m *Machine, e *models.Error) renderers {
	if m == nil {
		e.Errorf("No machine to render against")
		return nil
	}
	r := newRenderData(rt, m, t)
	return r.makeRenderers(e)
}

var taskLockMap = map[string][]string{
	"get":    []string{"templates", "tasks"},
	"create": []string{"stages", "templates", "tasks", "bootenvs"},
	"update": []string{"stages", "templates", "tasks", "bootenvs"},
	"patch":  []string{"stages", "templates", "tasks", "bootenvs"},
	"delete": []string{"stages", "tasks", "machines"},
}

func (t *Task) Locks(action string) []string {
	return taskLockMap[action]
}
