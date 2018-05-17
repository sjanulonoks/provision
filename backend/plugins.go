package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Plugin represents a single instance of a running plugin.
// This contains the configuration need to start this plugin instance.
type Plugin struct {
	*models.Plugin
	// If there are any errors in the start-up process, they will be
	// available here.
	// read only: true
	validate
}

func (n *Plugin) SetReadOnly(b bool) {
	n.ReadOnly = b
}

func (n *Plugin) SaveClean() store.KeySaver {
	mod := *n.Plugin
	mod.ClearValidation()
	return toBackend(&mod, n.rt)
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

func (n *Plugin) Prefix() string {
	return "plugins"
}

func (n *Plugin) Key() string {
	return n.Name
}

func (n *Plugin) New() store.KeySaver {
	res := &Plugin{Plugin: &models.Plugin{}}
	if n.Plugin != nil && n.ChangeForced() {
		res.ForceChange()
	}
	res.rt = n.rt
	return res
}

func (n *Plugin) Validate() {
	n.Plugin.Validate()
	n.AddError(index.CheckUnique(n, n.rt.stores("plugins").Items()))
	ValidateParams(n.rt, n, n.Params, n.rt.PrivateKeyFor(n))
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
	defer func() { n.rt = nil }()
	n.Fill()
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
	"get":     {"plugins", "params", "profiles"},
	"create":  {"plugins", "params"},
	"update":  {"plugins", "params"},
	"patch":   {"plugins", "params"},
	"delete":  {"plugins", "params"},
	"actions": {"plugins", "profiles", "params"},
}

func (n *Plugin) Locks(action string) []string {
	return pluginLockMap[action]
}
