package backend

import (
	"sync"
	"text/template"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

// Task is a thing that can run on a Machine.
//
// swagger:model
type Task struct {
	// Name is the name of this Task.  Task names must be globally unique
	//
	// required: true
	Name string
	// Description is a one-line description of this Task.
	Description string
	// Documentation should describe in detail what this task should do on a machine.
	Documentation string
	// Templates lists the templates that need to be rendered for the Task.
	//
	// required: true
	Templates []TemplateInfo
	// RequiredParams is the list of parameters that are required to be present on
	// Machine.Params or in a profile attached to the machine.
	//
	// required: true
	RequiredParams []string
	// OptionalParams are extra optional parameters that a template rendered for
	// the Task may use.
	//
	// required: true
	OptionalParams []string
	p              *DataTracker
	rootTemplate   *template.Template
	tmplMux        sync.Mutex
}

func AsTask(o store.KeySaver) *Task {
	return o.(*Task)
}

func AsTasks(o []store.KeySaver) []*Task {
	res := make([]*Task, len(o))
	for i := range o {
		res[i] = AsTask(o[i])
	}
	return res
}

func (t *Task) Backend() store.SimpleStore {
	return t.p.getBackend(t)
}

func (t *Task) Prefix() string {
	return "tasks"
}

func (t *Task) Key() string {
	return t.Name
}

func (t *Task) New() store.KeySaver {
	res := &Task{Name: t.Name, p: t.p}
	return store.KeySaver(res)
}

func (d *DataTracker) NewTask() *Task {
	return &Task{p: d}
}

func (t *Task) setDT(dp *DataTracker) {
	t.p = dp
}

func (t *Task) List() []*Task {
	return AsTasks(t.p.FetchAll(t))
}

func (t *Task) Indexes() map[string]index.Maker {
	fix := AsTask
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Name": index.Make(
			true,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).Name < fix(j).Name },
			func(ref store.KeySaver) (gte, gt index.Test) {
				refName := fix(ref).Name
				return func(s store.KeySaver) bool {
						return fix(s).Name >= refName
					},
					func(s store.KeySaver) bool {
						return fix(s).Name > refName
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Task{Name: s}, nil
			}),
	}
}

func (t *Task) genRoot(common *template.Template, e *Error) *template.Template {
	return MergeTemplates(common, t.Templates, e)
}

func (t *Task) OnLoad() error {
	e := &Error{o: t}
	t.tmplMux.Lock()
	defer t.tmplMux.Unlock()
	t.p.tmplMux.Lock()
	defer t.p.tmplMux.Unlock()
	t.rootTemplate = t.genRoot(t.p.rootTemplate, e)
	return e.OrNil()
}

func (t *Task) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: t}
	t.p.tmplMux.Lock()
	defer t.p.tmplMux.Unlock()
	t.tmplMux.Lock()
	defer t.tmplMux.Unlock()
	root := t.genRoot(t.p.rootTemplate, e)
	if !e.ContainsError() {
		t.rootTemplate = root
	}
	return e.OrNil()
}

type taskHaver interface {
	store.KeySaver
	HasTask(string) bool
}

func (t *Task) BeforeDelete() error {
	e := &Error{Code: 409, Type: StillInUseError, o: t}
	objs, unlocker := t.p.lockEnts("profiles", "machines", "bootenvs")
	defer unlocker()
	for i := range objs {
		for j := range objs[i].d {
			thing := objs[i].d[j].(taskHaver)
			if thing.HasTask(t.Name) {
				e.Errorf("%s:%s still uses %s", thing.Prefix(), thing.Key(), t.Name)
			}
		}
	}
	return e.OrNil()
}

func (t *Task) renderInfo() ([]TemplateInfo, []string) {
	return t.Templates, t.RequiredParams
}

func (t *Task) templates() *template.Template {
	return t.rootTemplate
}

func (t *Task) Render(m *Machine, e *Error) renderers {
	if m == nil {
		e.Errorf("No machine to render against")
		return nil
	}
	r := newRenderData(t.p, m, t)
	return r.makeRenderers(e)
}
