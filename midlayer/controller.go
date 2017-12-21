package midlayer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	"github.com/gin-gonic/gin"
)

type RunningPlugin struct {
	Plugin   *backend.Plugin
	Provider *models.PluginProvider
	Client   *PluginClient
}

type PluginController struct {
	lock               sync.Mutex
	AvailableProviders map[string]*models.PluginProvider
	runningPlugins     map[string]*RunningPlugin
	dt                 *backend.DataTracker
	pluginDir          string
	done               chan bool
	finished           chan bool
	events             chan *models.Event
	publishers         *backend.Publishers
	MachineActions     *MachineActions
	apiPort            int
}

func InitPluginController(pluginDir string, dt *backend.DataTracker, pubs *backend.Publishers, apiPort int) (pc *PluginController, err error) {
	pc = &PluginController{pluginDir: pluginDir, dt: dt, publishers: pubs,
		AvailableProviders: make(map[string]*models.PluginProvider, 0), apiPort: apiPort,
		runningPlugins: make(map[string]*RunningPlugin, 0)}

	pc.MachineActions = NewMachineActions()
	pubs.Add(pc)

	pc.done = make(chan bool)
	pc.finished = make(chan bool)
	pc.events = make(chan *models.Event, 1000)

	go func() {
		done := false
		for !done {
			select {
			case event := <-pc.events:
				pc.lock.Lock()
				ref := &backend.Plugin{}
				rt := pc.dt.Request(pc.dt.Logger, ref.Locks("get")...)
				rt.Do(func(d backend.Stores) {
					ref2 := rt.Find(ref.Prefix(), event.Key)
					switch event.Action {
					case "create":
						// May be deleted before we get here.
						if ref2 != nil {
							pc.startPlugin(rt, ref2.(*backend.Plugin))
						}
					case "save", "update":
						// May be deleted before we get here.
						if ref2 != nil {
							pc.restartPlugin(rt, ref2.(*backend.Plugin))
						}
					case "delete":
						pc.stopPlugin(event.Object.(*backend.Plugin))
					default:
						rt.Infof("debugPlugins", "internal event:", event)
					}
				})
				pc.lock.Unlock()
			case <-pc.done:
				done = true
			}
		}
		pc.finished <- true
	}()

	err = pc.WalkPluginDir()
	return
}

func (pc *PluginController) WalkPluginDir() error {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	// Walk plugin dir contents with lock
	files, err := ioutil.ReadDir(pc.pluginDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		pc.dt.Infof("debugPlugins", "Walk plugin dir: %s\n", f.Name())
		err = pc.importPluginProvider(f.Name())
		if err != nil {
			return err
		}
	}
	return nil
}

func (pc *PluginController) walkPlugins(provider string) (err error) {
	// Walk all plugin objects from dt.
	ref := &backend.Plugin{}
	rt := pc.dt.Request(pc.dt.Logger, ref.Locks("get")...)
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
				pc.startPlugin(rt, plugin)
			}
		}
	})
	return
}

func (pc *PluginController) Shutdown(ctx context.Context) error {
	pc.done <- true
	<-pc.finished
	return nil
}

func (pc *PluginController) Publish(e *models.Event) error {
	if e.Type != "plugins" {
		return nil
	}
	pc.events <- e
	return nil
}

// This never gets unloaded.
func (pc *PluginController) Reserve() error {
	return nil
}
func (pc *PluginController) Release() {}
func (pc *PluginController) Unload()  {}

func (pc *PluginController) GetPluginProvider(name string) *models.PluginProvider {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	if pp, ok := pc.AvailableProviders[name]; !ok {
		return nil
	} else {
		return pp
	}
}

func (pc *PluginController) GetPluginProviders() []*models.PluginProvider {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	// get the list of keys and sort them
	keys := []string{}
	for key := range pc.AvailableProviders {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	answer := []*models.PluginProvider{}
	for _, key := range keys {
		answer = append(answer, pc.AvailableProviders[key])
	}
	return answer
}

func (pc *PluginController) startPlugin(rt *backend.RequestTracker, plugin *backend.Plugin) {
	pc.dt.Infof("debugPlugins", "Starting plugin: %s(%s)\n", plugin.Name, plugin.Provider)
	if _, ok := pc.runningPlugins[plugin.Name]; ok {
		pc.dt.Infof("debugPlugins", "Already started plugin: %s(%s)\n", plugin.Name, plugin.Provider)
	}
	pp, ok := pc.AvailableProviders[plugin.Provider]
	if ok {
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

		if len(errors) == 0 {
			ppath := pc.pluginDir + "/" + pp.Name
			thingee, err := NewPluginClient(plugin.Name, pc.dt, pc.apiPort, ppath, plugin.Params)
			if err == nil {
				rp := &RunningPlugin{Plugin: plugin, Client: thingee, Provider: pp}
				if pp.HasPublish {
					pc.publishers.Add(thingee)
				}
				for _, aa := range pp.AvailableActions {
					aa.Provider = pp.Name
					pc.MachineActions.Add(aa, rp)
				}
				pc.runningPlugins[plugin.Name] = rp
			} else {
				errors = append(errors, err.Error())
			}
		}

		if plugin.PluginErrors == nil {
			plugin.PluginErrors = []string{}
		}
		if len(plugin.PluginErrors) != len(errors) {
			plugin.PluginErrors = errors
			rt.Update(plugin)
		}
		pc.publishers.Publish("plugin", "started", plugin.Name, plugin)
		pc.dt.Infof("debugPlugins", "Starting plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
	} else {
		pc.dt.Infof("debugPlugins", "Starting plugin: %s(%s) missing provider\n", plugin.Name, plugin.Provider)
		if plugin.PluginErrors == nil || len(plugin.PluginErrors) == 0 {
			plugin.Errors = []string{fmt.Sprintf("Missing Plugin Provider: %s", plugin.Provider)}
			rt.Update(plugin)
		}
	}
}

func (pc *PluginController) stopPlugin(plugin *backend.Plugin) {
	rp, ok := pc.runningPlugins[plugin.Name]
	if ok {
		pc.dt.Infof("debugPlugins", "Stopping plugin: %s(%s)\n", plugin.Name, plugin.Provider)
		delete(pc.runningPlugins, plugin.Name)

		if rp.Provider.HasPublish {
			pc.publishers.Remove(rp.Client)
		}
		for _, aa := range rp.Provider.AvailableActions {
			pc.MachineActions.Remove(aa)
		}
		rp.Client.Stop()
		pc.dt.Infof("debugPlugins", "Stoping plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
		pc.publishers.Publish("plugin", "stopped", plugin.Name, plugin)
	}
}

func (pc *PluginController) restartPlugin(rt *backend.RequestTracker, plugin *backend.Plugin) {
	rt.Infof("debugPlugins", "Restarting plugin: %s(%s)\n", plugin.Name, plugin.Provider)
	pc.stopPlugin(plugin)
	pc.startPlugin(rt, plugin)
	rt.Infof("debugPlugins", "Restarting plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
}

func (pc *PluginController) buildNewStore(content *models.Content) (newStore store.Store, err error) {
	filename := fmt.Sprintf("memory:///")

	newStore, err = store.Open(filename)
	if err != nil {
		return
	}

	if md, ok := newStore.(store.MetaSaver); ok {
		data := map[string]string{
			"Name":        content.Meta.Name,
			"Source":      content.Meta.Source,
			"Description": content.Meta.Description,
			"Version":     content.Meta.Version,
			"Type":        content.Meta.Type,
		}
		md.SetMetaData(data)
	}

	for prefix, objs := range content.Sections {
		var sub store.Store
		sub, err = newStore.MakeSub(prefix)
		if err != nil {
			return
		}

		for k, obj := range objs {
			err = sub.Save(k, obj)
			if err != nil {
				return
			}
		}
	}

	return
}

func forceParamRemoval(d *DataStack, l store.Store) error {
	toRemove := [][]string{}
	layer0 := d.Layers()[0]
	lSubs := l.Subs()
	dSubs := layer0.Subs()
	for k, v := range lSubs {
		dSub := dSubs[k]
		if dSub == nil {
			continue
		}
		lKeys, _ := v.Keys()
		for _, key := range lKeys {
			var dItem interface{}
			var lItem interface{}
			if err := dSub.Load(key, &dItem); err != nil {
				continue
			}
			if err := v.Load(key, &lItem); err != nil {
				return err
			}
			toRemove = append(toRemove, []string{k, key})
		}
	}
	for _, item := range toRemove {
		dSub := d.Subs()[item[0]]
		dSub.Remove(item[1])
	}
	return nil
}

// Try to add to available - Must lock before calling
func (pc *PluginController) importPluginProvider(provider string) error {
	pc.dt.Infof("debugPlugins", "Importing plugin provider: %s\n", provider)
	out, err := exec.Command(pc.pluginDir+"/"+provider, "define").Output()
	if err != nil {
		pc.dt.Infof("debugPlugins", "Skipping %s because %s\n", provider, err)
	} else {
		pp := &models.PluginProvider{}
		err = json.Unmarshal(out, pp)
		if err != nil {
			pc.dt.Infof("debugPlugins", "Skipping %s because of bad json: %s\n%s\n", provider, err, out)
		} else {
			skip := false
			pp.Fill()

			content := &models.Content{}
			content.Fill()

			if pp.Content != "" {
				codec := store.YamlCodec
				if err := codec.Decode([]byte(pp.Content), content); err != nil {
					return err
				}
			} else {
				content.Meta.Meta = pp.Meta
			}
			cName := pp.Name
			content.Meta.Name = cName

			if content.Meta.Version == "" || content.Meta.Version == "Unspecified" {
				content.Meta.Version = pp.Version
			}
			if content.Meta.Description == "" {
				content.Meta.Description = fmt.Sprintf("Content layer for %s plugin provider", pp.Name)
			}
			if content.Meta.Source == "" {
				content.Meta.Source = "FromPluginProvider"
			}
			content.Meta.Type = "plugin"

			// Merge in parameters if old plugin.
			if _, ok := content.Sections["params"]; !ok {
				content.Sections["params"] = models.Section{}
			}
			for _, p := range pp.Parameters {
				p.Fill()
				content.Sections["params"][p.Name] = p
			}

			if !skip {
				if ns, err := pc.buildNewStore(content); err != nil {
					pc.dt.Infof("debugPlugins", "Skipping %s because of bad store: %v\n", pp.Name, err)
					return err
				} else {
					err := func() error {
						_, unlocker := pc.dt.LockAll()
						defer unlocker()
						// Add replace the plugin content
						ds := pc.dt.Backend.(*DataStack)
						if nbs, hard, _ := ds.AddReplacePlugin(cName, ns, pc.dt.Logger, forceParamRemoval); hard != nil {
							pc.dt.Infof("debugPlugins", "Skipping %s because of bad store errors: %v\n", pp.Name, hard)
							return hard
						} else {
							pc.dt.ReplaceBackend(nbs)
						}
						return nil
					}()
					if err != nil {
						return err
					}
				}

				if _, ok := pc.AvailableProviders[pp.Name]; !ok {
					pc.dt.Infof("debugPlugins", "Adding plugin provider: %s\n", pp.Name)
					pp.Fill()
					pc.AvailableProviders[pp.Name] = pp
					for _, aa := range pp.AvailableActions {
						aa.Provider = pp.Name
					}
					pc.publishers.Publish("plugin_provider", "create", pp.Name, pp)
					return pc.walkPlugins(provider)
				} else {
					pc.dt.Infof("debugPlugins", "Already exists plugin provider: %s\n", pp.Name)
				}
			}
		}
	}
	return nil
}

// Try to stop using plugins and remove available - Must lock before calling
func (pc *PluginController) removePluginProvider(provider string) error {
	var name string
	for _, pp := range pc.AvailableProviders {
		if provider == pp.Name {
			name = pp.Name
			break
		}
	}
	if name != "" {
		remove := []*backend.Plugin{}
		for _, rp := range pc.runningPlugins {
			if rp.Provider.Name == name {
				remove = append(remove, rp.Plugin)
			}
		}
		ref := &backend.Plugin{}
		rt := pc.dt.Request(pc.dt.Logger, ref.Locks("get")...)
		rt.Do(func(d backend.Stores) {
			for _, p := range remove {
				pc.stopPlugin(p)
				ref2 := rt.Find(ref.Prefix(), p.Name)
				myPP := ref2.(*backend.Plugin)
				myPP.Errors = []string{fmt.Sprintf("Missing Plugin Provider: %s", provider)}
				rt.Update(myPP)
			}
		})

		pc.dt.Infof("debugPlugins", "Removing plugin provider: %s\n", name)
		pc.publishers.Publish("plugin_provider", "delete", name, pc.AvailableProviders[name])

		// Remove the plugin content
		func() {
			_, unlocker := pc.dt.LockAll()
			defer unlocker()
			ds := pc.dt.Backend.(*DataStack)
			if nbs, hard, _ := ds.RemovePlugin(name, pc.dt.Logger); hard != nil {
				fmt.Printf("Skipping removal of plugin content layer %s because of bad store errors: %v\n", name, hard)
				pc.dt.Infof("debugPlugins", "Skipping removal of plugin content layer %s because of bad store errors: %v\n", name, hard)
			} else {
				pc.dt.ReplaceBackend(nbs)
			}
		}()
		delete(pc.AvailableProviders, name)
	}

	return nil
}

func (pc *PluginController) UploadPlugin(c *gin.Context, fileRoot, name string) (*models.PluginProviderUploadInfo, *models.Error) {
	if err := os.MkdirAll(path.Join(fileRoot, `plugins`), 0755); err != nil {
		return nil, models.NewError("API_ERROR", http.StatusConflict,
			fmt.Sprintf("upload: unable to create plugins directory"))
	}
	var copied int64
	ctype := c.Request.Header.Get(`Content-Type`)
	switch strings.Split(ctype, "; ")[0] {
	case `application/octet-stream`:
		if c.Request.Body == nil {
			return nil, models.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: Unable to upload %s: missing body", name))
		}
	case `multipart/form-data`:
		header, err := c.FormFile("file")
		if err != nil {
			return nil, models.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: Failed to find multipart file: %v", err))
		}
		name = path.Base(header.Filename)
	default:
		return nil, models.NewError("API ERROR", http.StatusUnsupportedMediaType,
			fmt.Sprintf("upload: plugin_provider %s content-type %s is not application/octet-stream or multipart/form-data", name, ctype))
	}

	ppTmpName := path.Join(pc.pluginDir, fmt.Sprintf(`.%s.part`, path.Base(name)))
	ppName := path.Join(pc.pluginDir, path.Base(name))
	if _, err := os.Open(ppTmpName); err == nil {
		return nil, models.NewError("API ERROR", http.StatusConflict,
			fmt.Sprintf("upload: plugin_provider %s already uploading", name))
	}
	tgt, err := os.Create(ppTmpName)
	defer tgt.Close()
	if err != nil {
		return nil, models.NewError("API ERROR", http.StatusConflict,
			fmt.Sprintf("upload: Unable to upload %s: %v", name, err))
	}

	switch strings.Split(ctype, "; ")[0] {
	case `application/octet-stream`:
		copied, err = io.Copy(tgt, c.Request.Body)
		if err != nil {
			os.Remove(ppTmpName)
			return nil, models.NewError("API ERROR", http.StatusInsufficientStorage,
				fmt.Sprintf("upload: Failed to upload %s: %v", name, err))
		}
		if c.Request.ContentLength > 0 && copied != c.Request.ContentLength {
			os.Remove(ppTmpName)
			return nil, models.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: Failed to upload entire file %s: %d bytes expected, %d bytes received", name, c.Request.ContentLength, copied))
		}
	case `multipart/form-data`:
		header, _ := c.FormFile("file")
		file, err := header.Open()
		defer file.Close()
		copied, err = io.Copy(tgt, file)
		if err != nil {
			return nil, models.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: iso %s could not save", header.Filename))
		}
		file.Close()
	}
	tgt.Close()

	os.Remove(ppName)
	os.Rename(ppTmpName, ppName)
	os.Chmod(ppName, 0700)

	pc.lock.Lock()
	defer pc.lock.Unlock()
	// If it is here, remove it.
	pc.removePluginProvider(name)

	var berr *models.Error
	err = pc.importPluginProvider(name)
	if err != nil {
		berr = models.NewError("API ERROR", http.StatusBadRequest,
			fmt.Sprintf("Import plugin failed %s: %v", name, err))
	}
	return &models.PluginProviderUploadInfo{Path: name, Size: copied}, berr
}

func (pc *PluginController) RemovePlugin(name string) error {
	pluginProviderName := path.Join(pc.pluginDir, path.Base(name))
	if err := os.Remove(pluginProviderName); err != nil {
		return err
	}
	pc.lock.Lock()
	defer pc.lock.Unlock()
	return pc.removePluginProvider(name)
}
