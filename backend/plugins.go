package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Plugin represents a single instance of a running plugin.
// This contains the configuration need to start this plugin instance.
// swagger:model
type Plugin struct {
	*models.Plugin
	// If there are any errors in the start-up process, they will be
	// available here.
	// read only: true
	validate
	p *DataTracker
}

func (obj *Plugin) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Plugin) SaveClean() store.KeySaver {
	mod := *obj.Plugin
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func (n *Plugin) Indexes() map[string]index.Maker {
	fix := AsPlugin
	res := index.MakeBaseIndexes(n)
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
			plugin := fix(n.New())
			plugin.Name = s
			return plugin, nil
		})
	res["Provider"] = index.Make(
		false,
		"string",
		func(i, j models.Model) bool { return fix(i).Provider < fix(j).Provider },
		func(ref models.Model) (gte, gt index.Test) {
			refProvider := fix(ref).Provider
			return func(s models.Model) bool {
					return fix(s).Provider >= refProvider
				},
				func(s models.Model) bool {
					return fix(s).Provider > refProvider
				}
		},
		func(s string) (models.Model, error) {
			plugin := fix(n.New())
			plugin.Provider = s
			return plugin, nil
		})
	return res
}

func (n *Plugin) Backend() store.Store {
	return n.p.getBackend(n)
}

func (n *Plugin) Prefix() string {
	return "plugins"
}

func (n *Plugin) Key() string {
	return n.Name
}

func (n *Plugin) GetParams(d Stores, _ bool) map[string]interface{} {
	m := n.Params
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

func (n *Plugin) SetParams(d Stores, values map[string]interface{}) error {
	n.Params = values
	e := &models.Error{Code: 422, Type: ValidationError, Model: n.Prefix(), Key: n.Key()}
	_, e2 := n.p.Save(d, n)
	e.AddError(e2)
	return e.HasError()
}

func (n *Plugin) GetParam(d Stores, key string, aggregate bool) (interface{}, bool) {
	mm := n.GetParams(d, aggregate)
	if v, found := mm[key]; found {
		return v, true
	}
	// Check the param itself
	if p := d("params").Find(key); p != nil && aggregate {
		param := p.(*Param)
		return param.DefaultValue()
	}
	return nil, false
}

func (n *Plugin) SetParam(d Stores, key string, val interface{}) error {
	n.Params[key] = val
	e := &models.Error{Code: 422, Type: ValidationError, Model: n.Prefix(), Key: n.Key()}
	_, e2 := n.p.Save(d, n)
	e.AddError(e2)
	return e.HasError()
}

func (n *Plugin) New() store.KeySaver {
	res := &Plugin{Plugin: &models.Plugin{}}
	if n.Plugin != nil && n.ChangeForced() {
		res.ForceChange()
	}
	res.p = n.p
	return res
}

func (n *Plugin) setDT(p *DataTracker) {
	n.p = p
}

func (n *Plugin) Validate() {
	n.Plugin.Validate()
	n.AddError(index.CheckUnique(n, n.stores("plugins").Items()))
	n.SetValid()
	n.SetAvailable()
}

func (n *Plugin) BeforeSave() error {
	n.Validate()
	if !n.Useable() {
		return n.MakeError(422, ValidationError, n)
	}
	return nil
}

func (n *Plugin) OnLoad() error {
	n.stores = func(ref string) *Store {
		return n.p.objs[ref]
	}
	defer func() { n.stores = nil }()
	return n.BeforeSave()
}

func AsPlugin(o models.Model) *Plugin {
	return o.(*Plugin)
}

func AsPlugins(o []models.Model) []*Plugin {
	res := make([]*Plugin, len(o))
	for i := range o {
		res[i] = AsPlugin(o[i])
	}
	return res
}

var pluginLockMap = map[string][]string{
	"get":    []string{"plugins", "params"},
	"create": []string{"plugins", "params"},
	"update": []string{"plugins", "params"},
	"patch":  []string{"plugins", "params"},
	"delete": []string{"plugins", "params"},
}

func (m *Plugin) Locks(action string) []string {
	return pluginLockMap[action]
}
