package midlayer

import (
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
)

const (
	PLUGIN_REQUESTED int = 0
	PLUGIN_CREATED   int = 1
	PLUGIN_STARTED   int = 2
	PLUGIN_CONFIGED  int = 3
	PLUGIN_STOPPED   int = 4
	PLUGIN_REMOVED   int = 5
	PLUGIN_FAILED    int = 7
)

type RunningPlugin struct {
	Plugin   *models.Plugin
	Provider *models.PluginProvider
	Client   *PluginClient
	state    int
}

/*
 * There are 6 backend events the drive plugin operations
 *   plugin.create  - Add plugin to running plugins list as Requested - generate start event
 *   plugin.update  - Difference plugins to see if restart is needed - generate stop/start events
 *   plugin.save    - Difference plugins to see if restart is needed - generate stop/start events
 *   plugin.delete  - Remove plugin from system - Generate stop and remove event
 *
 *   plugin_provider.create  - Plugin Provider created by frontend or startup - generate create and start events
 *   plugin_provider.delete  - Plugin Provider deleted by frontend - generate stop events
 *
 * There are 3 internal events
 *   plugin.start   - Plugin should be started
 *   plugin.stop    - Plugin should be stopped
 *   plugin.remove  - Plugin should be removed
 */

/*
 * This a separate go func for handling starting and stopping plugins.
 *
 * All actions are handled from this thread.
 */
func (pc *PluginController) handleEvent(event *models.Event) {
	switch event.Type {
	case "plugins", "plugin":
		plugin, _ := event.Model()
		switch event.Action {
		// API Actions
		case "create":
			pc.createPlugin(plugin)
		case "save", "update":
			pc.restartPlugin(plugin)
		case "delete":
			pc.deletePlugin(plugin)

		// State Actions
		case "start":
			pc.startPlugin(plugin)
		case "config":
			pc.configPlugin(plugin)
		case "stop":
			pc.stopPlugin(plugin)
		case "remove":
			pc.removePlugin(plugin)
		}
	// These generate events which get handled above.
	case "plugin_providers", "plugin_provider":
		switch event.Action {
		case "create":
			pc.allPlugins(event.Key, "start")
		case "delete":
			pc.allPlugins(event.Key, "stop")
		}
	}
}

func (pc *PluginController) StartPlugins() {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	// Get all the plugins that have this as provider
	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		var idx *index.Index
		idx, err := index.All([]index.Filter{index.Native()}...)(&d(ref.Prefix()).Index)
		if err != nil {
			return
		}
		arr := idx.Items()
		for _, res := range arr {
			plugin := res.(*backend.Plugin)
			// If we don't know about this plugin yet, create it on the running list
			if _, ok := pc.runningPlugins[plugin.Name]; !ok {
				rt.PublishEvent(models.EventFor(plugin, "create"))
			}
			rt.PublishEvent(models.EventFor(plugin, "start"))
		}
	})
	return
}

func (pc *PluginController) allPlugins(provider, action string) (err error) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	// Get all the plugins that have this as provider
	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		var idx *index.Index
		idx, err = index.All([]index.Filter{index.Native()}...)(&d(ref.Prefix()).Index)
		if err != nil {
			return
		}
		arr := idx.Items()
		for _, res := range arr {
			plugin := res.(*backend.Plugin)
			if plugin.Provider == provider {
				// If we don't know about this plugin yet, create it on the running list
				if _, ok := pc.runningPlugins[plugin.Name]; !ok {
					if action == "start" {
						rt.PublishEvent(models.EventFor(plugin, "create"))
					}
				}
				if action == "stop" {
					rt.PublishEvent(models.EventFor(plugin, "stop"))
				}
				// We even start on stop to get the error in place.
				rt.PublishEvent(models.EventFor(plugin, "start"))
			}
		}
	})
	return
}

func (pc *PluginController) createPlugin(mp models.Model) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	plugin := mp.(*models.Plugin)

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)

	if r, ok := pc.runningPlugins[plugin.Name]; ok && r.state == PLUGIN_CREATED {
		pc.Infof("Already created plugin: %s\n", plugin.Name)
		r.Plugin = plugin
	} else if ok {
		pc.Errorf("Plugin is already created and beyond created.  Wrong: %v", r.Plugin)
		r.Plugin = plugin
	} else {
		pc.runningPlugins[plugin.Name] = &RunningPlugin{Plugin: plugin, state: PLUGIN_CREATED}
	}
	rt.PublishEvent(models.EventFor(plugin, "start"))
}

// Must be called under rt.Do()
func validateParameters(rt *backend.RequestTracker, pp *models.PluginProvider, plugin *models.Plugin) []string {
	errors := []string{}
	for _, parmName := range pp.RequiredParams {
		obj, ok := plugin.Params[parmName]
		if !ok {
			errors = append(errors, fmt.Sprintf("Missing required parameter: %s", parmName))
		} else {
			pobj := rt.Find("params", parmName)
			if pobj != nil {
				rp := pobj.(*backend.Param)

				if ev := rp.ValidateValue(obj); ev != nil {
					errors = append(errors, ev.Error())
				}
			}
		}
	}
	for _, parmName := range pp.OptionalParams {
		obj, ok := plugin.Params[parmName]
		if ok {
			pobj := rt.Find("params", parmName)
			if pobj != nil {
				rp := pobj.(*backend.Param)

				if ev := rp.ValidateValue(obj); ev != nil {
					errors = append(errors, ev.Error())
				}
			}
		}
	}
	return errors
}
func (pc *PluginController) startPlugin(mp models.Model) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	plugin := mp.(*models.Plugin)

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		ref2 := rt.Find("plugins", plugin.Name)

		r, ok := pc.runningPlugins[plugin.Name]
		if !ok && ref2 == nil {
			// The plugin is deleted and not present.
			pc.Errorf("Plugin delete before starting. %v\n", plugin)
			return
		} else if !ok && ref2 != nil {
			pc.Infof("Plugin wants to be started, but isn't created, create it: %s\n", plugin.Name)
			rt.PublishEvent(models.EventFor(plugin, "create"))
			return
		} else if r.state == PLUGIN_STARTED {
			pc.Infof("Plugin is already started. Try to config it")
			rt.PublishEvent(models.EventFor(plugin, "config"))
		} else if r.state != PLUGIN_CREATED && r.state != PLUGIN_FAILED && r.state != PLUGIN_STOPPED {
			pc.Infof("Plugin is already past start.  just return")
			return
		}

		r.Plugin = plugin

		pp, ok := pc.AvailableProviders[plugin.Provider]
		if !ok {
			r.state = PLUGIN_FAILED
			pc.Errorf("Starting plugin: %s(%s) missing provider\n", plugin.Name, plugin.Provider)
			if plugin.PluginErrors == nil || len(plugin.PluginErrors) == 0 {
				plugin.PluginErrors = []string{fmt.Sprintf("Missing Plugin Provider: %s", plugin.Provider)}
				rt.Update(plugin)
			}
			return
		}

		r.Provider = pp

		errors := validateParameters(rt, pp, plugin)
		if len(errors) > 0 {
			r.state = PLUGIN_FAILED
			if plugin.PluginErrors == nil {
				plugin.PluginErrors = []string{}
			}
			if len(plugin.PluginErrors) != len(errors) {
				plugin.PluginErrors = errors
				rt.Update(plugin)
			}
			return
		}

		claims := backend.NewClaim(plugin.Name, "system", time.Hour*1000000).
			Add("*", "*", "*").
			AddSecrets("", "", "")
		token, _ := rt.SealClaims(claims)
		ppath := pc.pluginDir + "/" + pp.Name
		thingee, err := NewPluginClient(
			pc,
			pc.pluginCommDir,
			plugin.Name,
			pc.Logger.Fork().SetService(plugin.Name),
			rt.ApiURL(net.ParseIP("0.0.0.0")),
			rt.FileURL(net.ParseIP("0.0.0.0")),
			token,
			ppath, plugin.Params)

		if err != nil {
			if len(r.Plugin.PluginErrors) == 0 {
				r.Plugin.PluginErrors = []string{err.Error()}
				rt.Update(r.Plugin)
			}
			r.state = PLUGIN_FAILED
		} else {
			r.Client = thingee
			r.state = PLUGIN_STARTED

			if len(r.Plugin.PluginErrors) > 0 {
				r.Plugin.PluginErrors = []string{}
				rt.Update(r.Plugin)
			}
			rt.PublishEvent(models.EventFor(r.Plugin, "config"))
		}
	})
}

func (pc *PluginController) configPlugin(mp models.Model) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)

	plugin := mp.(*models.Plugin)

	r, ok := pc.runningPlugins[plugin.Name]
	if !ok {
		// The plugin is deleted and not present.
		pc.Errorf("Plugin delete before config. %v\n", plugin.Name)
		return
	} else if r.state == PLUGIN_CONFIGED {
		pc.Infof("Plugin is already configed. Done!")
		return
	} else if r.state != PLUGIN_STARTED {
		pc.Infof("Plugin is already isn't started.  Start over")
		rt.PublishEvent(models.EventFor(r.Plugin, "config"))
		return
	}

	pc.lock.Unlock()

	// Configure the plugin
	pc.Debugf("Config Plugin: %s\n", plugin)
	terr := r.Client.Config(plugin.Params)

	pc.lock.Lock()

	if terr != nil {
		r.Client.Debugf("Stop Plugin: %s Error: %v\n", plugin, terr)
		r.Client.Stop()
		r.Client = nil
		r.state = PLUGIN_FAILED
		rt.Do(func(d backend.Stores) {
			ref2 := rt.Find("plugins", plugin.Name)
			if ref2 == nil {
				return
			}
			r.Plugin = plugin

			if len(r.Plugin.PluginErrors) == 0 {
				r.Plugin.PluginErrors = []string{terr.Error()}
				rt.Update(r.Plugin)
			}
		})
		return
	}

	r.state = PLUGIN_CONFIGED
	if r.Provider.HasPublish {
		pc.publishers.Add(r.Client)
	}
	for i, _ := range r.Provider.AvailableActions {
		r.Provider.AvailableActions[i].Fill()
		r.Provider.AvailableActions[i].Provider = r.Provider.Name
		pc.Actions.Add(r.Provider.AvailableActions[i], r)
	}
	rt.Publish("plugin", "configed", plugin.Name, plugin)
}

func (pc *PluginController) stopPlugin(mp models.Model) {
	plugin := mp.(*models.Plugin)

	pc.lock.Lock()
	defer pc.lock.Unlock()

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)

	rp, ok := pc.runningPlugins[plugin.Name]
	if !ok || rp.state == PLUGIN_REMOVED || rp.state == PLUGIN_STOPPED {
		// If we've missing, been removed, or stopped, then done
		return
	}
	if rp.state == PLUGIN_STARTED || rp.state == PLUGIN_CONFIGED {
		plugin := rp.Plugin
		rt.Infof("Stopping plugin: %s(%s)\n", plugin.Name, plugin.Provider)

		if rp.Provider.HasPublish {
			rt.Debugf("Remove publisher: %s(%s)\n", plugin.Name, plugin.Provider)
			pc.publishers.Remove(rp.Client)
		}
		for _, aa := range rp.Provider.AvailableActions {
			rt.Debugf("Remove actions: %s(%s,%s)\n", plugin.Name, plugin.Provider, aa.Command)
			pc.Actions.Remove(aa, rp)
		}
		pc.lock.Unlock()

		rt.Debugf("Drain executable: %s(%s)\n", plugin.Name, plugin.Provider)
		rp.Client.Unload()
		rt.Debugf("Stop executable: %s(%s)\n", plugin.Name, plugin.Provider)
		rp.Client.Stop()
		rt.Infof("Stopping plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
		rt.Publish("plugin", "stopped", plugin.Name, plugin)

		pc.lock.Lock()
		rp.state = PLUGIN_STOPPED
		rp.Client = nil
	} else {
		pc.Infof("Plugin should be started before stopping!! %v\n", rp.Plugin)
		rp.state = PLUGIN_STOPPED
	}
}

func (pc *PluginController) removePlugin(mp models.Model) {
	plugin := mp.(*models.Plugin)

	pc.lock.Lock()
	defer pc.lock.Unlock()

	rp, ok := pc.runningPlugins[plugin.Name]
	if !ok {
		// If Already gone.
		return
	}
	if rp.state != PLUGIN_STOPPED {
		pc.Errorf("Plugin should be stopped before removing!! %v\n", rp.Plugin)
	}
	rp.state = PLUGIN_REMOVED
	delete(pc.runningPlugins, plugin.Name)
}

/*
 * We've received an update/save to the plugin.  Figure out if
 * we need to stop and start the plugin because of changes.
 */
func (pc *PluginController) restartPlugin(mp models.Model) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	plugin := mp.(*models.Plugin)

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		ref2 := rt.Find(ref.Prefix(), plugin.Name)
		// May be deleted before we get here. An event will be around to remove it
		if ref2 != nil {
			p := ref2.(*backend.Plugin)

			if rp, ok := pc.runningPlugins[plugin.Name]; !ok {
				// We did fine our plugin in the list.  Send a create event (which will send a start event)
				// Just in case, we have a race on startup and somebody making changes.  Speed the process along.
				rt.PublishEvent(models.EventFor(p, "create"))
			} else {
				oldP := rp.Plugin
				doit := false
				if p.Description != oldP.Description {
					doit = true
				}
				if p.Provider != oldP.Provider {
					doit = true
				}
				if !reflect.DeepEqual(p.Params, oldP.Params) {
					doit = true
				}
				if doit {
					rt.PublishEvent(models.EventFor(p, "stop"))
					rt.PublishEvent(models.EventFor(p, "start"))
				}
			}
		}
	})
}

/*
 * We've received a delete to the plugin.
 */
func (pc *PluginController) deletePlugin(mp models.Model) {
	plugin := mp.(*models.Plugin)

	ref := &backend.Plugin{}
	rt := pc.Request(ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		ref2 := rt.Find(ref.Prefix(), plugin.Name)
		// May be deleted before we get here. An event will be around to remove it
		if ref2 != nil {
			p := ref2.(*backend.Plugin)
			rt.PublishEvent(models.EventFor(p, "stop"))
			rt.PublishEvent(models.EventFor(p, "remove"))
		}
	})
}
