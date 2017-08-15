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

func (n *Plugin) Indexes() map[string]index.Maker {
	fix := AsPlugin
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
				plugin := &Plugin{}
				plugin.Name = s
				return plugin, nil
			}),
		"Provider": index.Make(
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
				plugin := &Plugin{}
				plugin.Provider = s
				return plugin, nil
			}),
	}
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

func (n *Plugin) AuthKey() string {
	return n.Key()
}

func (n *Plugin) GetParams() map[string]interface{} {
	m := n.Params
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

func (n *Plugin) SetParams(d Stores, values map[string]interface{}) error {
	n.Params = values
	e := &models.Error{Code: 422, Type: ValidationError, Object: n}
	_, e2 := n.p.Save(d, n)
	e.AddError(e2)
	return e.HasError()
}

func (n *Plugin) GetParam(d Stores, key string, searchProfiles bool) (interface{}, bool) {
	mm := n.GetParams()
	if v, found := mm[key]; found {
		return v, true
	}
	return nil, false
}

func (n *Plugin) New() store.KeySaver {
	res := &Plugin{Plugin: &models.Plugin{}}
	return res
}

func (n *Plugin) setDT(p *DataTracker) {
	n.p = p
}

func (n *Plugin) Validate() {
	n.AddError(index.CheckUnique(n, n.stores("plugins").Items()))
	if n.Provider == "" {
		n.Errorf("Plugin %s must have a provider", n.Name)
	}
	n.SetValid()
	n.SetAvailable()
}

func (n *Plugin) BeforeSave() error {
	if !n.Useable() {
		return n.MakeError(422, ValidationError, n)
	}
	return nil
}

func (n *Plugin) OnLoad() error {
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
