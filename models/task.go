package models

// Task is a thing that can run on a Machine.
//
// swagger:model
type Task struct {
	Validation
	Access
	Meta
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
}

func (t *Task) GetMeta() Meta {
	return t.Meta
}

func (t *Task) SetMeta(d Meta) {
	t.Meta = d
}

func (t *Task) GetDocumentation() string {
	return t.Documentation
}

func (t *Task) Validate() {
	t.AddError(ValidName("Invalid Name", t.Name))

	for _, p := range t.RequiredParams {
		t.AddError(ValidParamName("Invalid Required Param", p))
	}
	for _, p := range t.OptionalParams {
		t.AddError(ValidParamName("Invalid Optional Param", p))
	}
	for _, tt := range t.Templates {
		if tt.ID != "" {
			t.AddError(ValidName("Invalid Template ID", tt.ID))
		}
	}
}

func (t *Task) Prefix() string {
	return "tasks"
}

func (t *Task) Key() string {
	return t.Name
}

func (t *Task) KeyName() string {
	return "Name"
}

func (t *Task) Fill() {
	t.Validation.fill()
	if t.Meta == nil {
		t.Meta = Meta{}
	}
	if t.Templates == nil {
		t.Templates = []TemplateInfo{}
	}
	if t.RequiredParams == nil {
		t.RequiredParams = []string{}
	}
	if t.OptionalParams == nil {
		t.OptionalParams = []string{}
	}
}

func (t *Task) AuthKey() string {
	return t.Key()
}

func (t *Task) SliceOf() interface{} {
	s := []*Task{}
	return &s
}

func (t *Task) ToModels(obj interface{}) []Model {
	items := obj.(*[]*Task)
	res := make([]Model, len(*items))
	for i, item := range *items {
		res[i] = Model(item)
	}
	return res
}

func (t *Task) SetName(n string) {
	t.Name = n
}

func (t *Task) CanHaveActions() bool {
	return true
}
