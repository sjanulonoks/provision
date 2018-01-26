package midlayer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/digitalrebar/logger"
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
	logger.Logger
	lock               sync.Mutex
	AvailableProviders map[string]*models.PluginProvider
	runningPlugins     map[string]*RunningPlugin
	dt                 *backend.DataTracker
	pluginDir          string
	pluginCommDir      string
	done               chan bool
	finished           chan bool
	events             chan *models.Event
	publishers         *backend.Publishers
	Actions            *Actions
}

func InitPluginController(pluginDir, pluginCommDir string, dt *backend.DataTracker, pubs *backend.Publishers) (pc *PluginController, err error) {
	dt.Logger.Debugf("Starting Plugin Controller\n")
	pc = &PluginController{
		Logger:             dt.Logger.Switch("plugin"),
		pluginDir:          pluginDir,
		pluginCommDir:      pluginCommDir,
		dt:                 dt,
		publishers:         pubs,
		AvailableProviders: make(map[string]*models.PluginProvider, 0),
		runningPlugins:     make(map[string]*RunningPlugin, 0)}

	pc.Actions = NewActions()
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
				rt := pc.dt.Request(pc.Logger, ref.Locks("get")...)
				rt.Do(func(d backend.Stores) {
					ref2 := rt.Find(ref.Prefix(), event.Key)
					switch event.Action {
					case "create":
						// May be deleted before we get here.
						if ref2 != nil {
							rt.Debugf("handling plugin create:", event)
							pc.startPlugin(rt, ref2.(*backend.Plugin))
						}
					case "save", "update":
						// May be deleted before we get here.
						if ref2 != nil {
							rt.Debugf("handling plugin save/update:", event)
							pc.restartPlugin(rt, ref2.(*backend.Plugin))
						}
					case "delete":
						rt.Debugf("handling plugin delete:", event)
						pc.stopPlugin(event.Object.(*backend.Plugin).Name)
					default:
						rt.Infof("internal event:", event)
					}
				})
				pc.lock.Unlock()
			case <-pc.done:
				done = true
			}
		}
		pc.finished <- true
	}()
	dt.Logger.Debugf("Returning Plugin Controller: %v\n", err)
	return
}

func ReverseProxy(l logger.Logger, pluginCommDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		plugin := c.Param(`plugin`)
		socketPath := fmt.Sprintf("%s/%s.toPlugin", pluginCommDir, plugin)

		url, _ := url.Parse(fmt.Sprintf("http://unix/%s", socketPath))
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (pc *PluginController) StartController(apiGroup *gin.RouterGroup) error {
	pc.Debugf("Starting Start Plugin Controller:\n")

	apiGroup.Any("/plugin-apis/:plugin/*path", ReverseProxy(pc, pc.pluginCommDir))

	err := pc.walkPluginDir()
	pc.Debugf("Finishing Start Plugin Controller: %v\n", err)
	return err
}

func (pc *PluginController) walkPluginDir() error {
	pc.Tracef("walkPlugDir: started\n")
	pc.lock.Lock()
	defer pc.lock.Unlock()
	// Walk plugin dir contents with lock
	files, err := ioutil.ReadDir(pc.pluginDir)
	if err != nil {
		pc.Tracef("walkPlugDir: finished ReadDir error: %v\n", err)
		return err
	}
	for _, f := range files {
		pc.Debugf("Walk plugin importing: %s\n", f.Name())
		err = pc.importPluginProvider(f.Name())
		if err != nil {
			pc.Tracef("walkPlugDir: importing %s error: %v\n", f.Name(), err)
		}
	}
	pc.Tracef("walkPlugDir: finished\n")
	return nil
}

func (pc *PluginController) walkPlugins(provider string) (err error) {
	pc.Tracef("walkPlugins: started\n")
	// Walk all plugin objects from dt.
	ref := &backend.Plugin{}
	rt := pc.dt.Request(pc.Logger, ref.Locks("get")...)
	rt.Do(func(d backend.Stores) {
		var idx *index.Index
		idx, err = index.All([]index.Filter{index.Native()}...)(&d(ref.Prefix()).Index)
		if err != nil {
			pc.Tracef("walkPlugins: finished error: %v\n", err)
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
	pc.Tracef("walkPlugins: finished\n")
	return
}

func (pc *PluginController) Shutdown(ctx context.Context) error {
	pc.Debugf("Stopping plugin controller\n")
	for _, rp := range pc.runningPlugins {
		pc.Debugf("Stopping plugin: %s\n", rp.Plugin.Name)
		pc.stopPlugin(rp.Plugin.Name)
	}
	pc.Debugf("Stopping plugin gofuncs\n")
	pc.done <- true
	pc.Debugf("Waiting for gofuncs to finish\n")
	<-pc.finished
	pc.Debugf("All stopped\n")
	return nil
}

func (pc *PluginController) Publish(e *models.Event) error {
	if e.Type != "plugins" && e.Type != "plugin" {
		return nil
	}
	pc.NoPublish().Tracef("PluginController Publish Event stared: %v\n", e)
	pc.events <- e
	pc.NoPublish().Tracef("PluginController Publish Event finished: %v\n", e)
	return nil
}

// This never gets unloaded.
func (pc *PluginController) Reserve() error {
	return nil
}
func (pc *PluginController) Release() {}
func (pc *PluginController) Unload()  {}

func (pc *PluginController) GetPluginProvider(name string) *models.PluginProvider {
	pc.Tracef("Starting GetPluginProvider\n")
	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.Debugf("Getting plugin provider for %s\n", name)
	if pp, ok := pc.AvailableProviders[name]; !ok {
		pc.Tracef("Returning GetPluginProvider: null\n")
		return nil
	} else {
		pc.Tracef("Returning GetPluginProvider: <one>\n")
		return pp
	}
}

func (pc *PluginController) GetPluginProviders() []*models.PluginProvider {
	pc.Tracef("Starting GetPluginProviders\n")
	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.Debugf("Getting all plugin providers\n")
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
	pc.Tracef("Returning GetPluginProviders: %d\n", len(answer))
	return answer
}

func (pc *PluginController) startPlugin(rt *backend.RequestTracker, plugin *backend.Plugin) {
	pc.Infof("Starting plugin: %s(%s)\n", plugin.Name, plugin.Provider)
	if _, ok := pc.runningPlugins[plugin.Name]; ok {
		pc.Infof("Already started plugin: %s(%s)\n", plugin.Name, plugin.Provider)
		return
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
			if err == nil {
				rp := &RunningPlugin{Plugin: plugin, Client: thingee, Provider: pp}
				if pp.HasPublish {
					pc.publishers.Add(thingee)
				}
				for i, _ := range pp.AvailableActions {
					pp.AvailableActions[i].Fill()
					pp.AvailableActions[i].Provider = pp.Name
					pc.Actions.Add(pp.AvailableActions[i], rp)
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
		pc.Infof("Starting plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
	} else {
		pc.Errorf("Starting plugin: %s(%s) missing provider\n", plugin.Name, plugin.Provider)
		if plugin.PluginErrors == nil || len(plugin.PluginErrors) == 0 {
			plugin.PluginErrors = []string{fmt.Sprintf("Missing Plugin Provider: %s", plugin.Provider)}
			rt.Update(plugin)
		}
	}
}

func (pc *PluginController) stopPlugin(pluginName string) {
	rp, ok := pc.runningPlugins[pluginName]
	if ok {
		plugin := rp.Plugin
		pc.Infof("Stopping plugin: %s(%s)\n", plugin.Name, plugin.Provider)
		delete(pc.runningPlugins, plugin.Name)

		if rp.Provider.HasPublish {
			pc.Debugf("Remove publisher: %s(%s)\n", plugin.Name, plugin.Provider)
			pc.publishers.Remove(rp.Client)
		}
		for _, aa := range rp.Provider.AvailableActions {
			pc.Debugf("Remove actions: %s(%s,%s)\n", plugin.Name, plugin.Provider, aa.Command)
			pc.Actions.Remove(aa, rp)
		}
		pc.Debugf("Drain executable: %s(%s)\n", plugin.Name, plugin.Provider)
		rp.Client.Unload()
		pc.Debugf("Stop executable: %s(%s)\n", plugin.Name, plugin.Provider)
		rp.Client.Stop()
		pc.Infof("Stopping plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
		pc.publishers.Publish("plugin", "stopped", plugin.Name, plugin)
	}
}

func (pc *PluginController) restartPlugin(rt *backend.RequestTracker, plugin *backend.Plugin) {
	rt.Infof("Restarting plugin: %s(%s)\n", plugin.Name, plugin.Provider)
	pc.stopPlugin(plugin.Name)
	pc.startPlugin(rt, plugin)
	rt.Infof("Restarting plugin: %s(%s) complete\n", plugin.Name, plugin.Provider)
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
	pc.Infof("Importing plugin provider: %s\n", provider)
	out, err := exec.Command(pc.pluginDir+"/"+provider, "define").Output()
	if err != nil {
		pc.Errorf("Skipping %s because %s\n", provider, err)
		return fmt.Errorf("Skipping %s because %s\n", provider, err)
	} else {
		pp := &models.PluginProvider{}
		err = json.Unmarshal(out, pp)
		if err != nil {
			pc.Errorf("Skipping %s because of bad json: %s\n%s\n", provider, err, out)
			return fmt.Errorf("Skipping %s because of bad json: %s\n%s\n", provider, err, out)
		} else {
			skip := false
			pp.Fill()

			if pp.PluginVersion != 2 {
				pc.Errorf("Skipping %s because of bad version: %d\n", provider, pp.PluginVersion)
				return fmt.Errorf("Skipping %s because of bad version: %d\n", provider, pp.PluginVersion)
			}

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

			if !skip {
				pc.Tracef("Building new datastore for: %s\n", provider)
				if ns, err := pc.buildNewStore(content); err != nil {
					pc.Errorf("Skipping %s because of bad store: %v\n", pp.Name, err)
					return err
				} else {
					var err error
					pc.Tracef("Replacing new datastore for: %s\n", provider)
					rt := pc.dt.Request(pc.Logger)
					rt.AllLocked(func(d backend.Stores) {
						ds := pc.dt.Backend.(*DataStack)
						nbs, hard, _ := ds.AddReplacePlugin(cName, ns, pc.dt.Logger, forceParamRemoval)
						if hard != nil {
							rt.Errorf("Skipping %s because of bad store errors: %v\n", pp.Name, hard)
							err = hard
							return
						}
						pc.dt.ReplaceBackend(rt, nbs)
					})
					pc.Tracef("Completed replacing new datastore for: %s\n", provider)
					if err != nil {
						return err
					}
				}

				if _, ok := pc.AvailableProviders[pp.Name]; !ok {
					pc.Infof("Adding plugin provider: %s\n", pp.Name)
					pp.Fill()
					pc.AvailableProviders[pp.Name] = pp
					for _, aa := range pp.AvailableActions {
						aa.Provider = pp.Name
					}
					pc.publishers.Publish("plugin_provider", "create", pp.Name, pp)
					return pc.walkPlugins(provider)
				} else {
					pc.Infof("Already exists plugin provider: %s\n", pp.Name)
				}
			}
		}
	}
	return nil
}

// Try to stop using plugins and remove available - Must lock controller lock before calling
func (pc *PluginController) removePluginProvider(provider string) error {
	pc.Tracef("removePluginProvider Started: %s\n", provider)
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
				pc.stopPlugin(p.Name)
				ref2 := rt.Find(ref.Prefix(), p.Name)
				myPP := ref2.(*backend.Plugin)
				myPP.Errors = []string{fmt.Sprintf("Missing Plugin Provider: %s", provider)}
				rt.Update(myPP)
			}
		})

		pc.Infof("Removing plugin provider: %s\n", name)
		pc.publishers.Publish("plugin_provider", "delete", name, pc.AvailableProviders[name])

		// Remove the plugin content
		rt.AllLocked(func(d backend.Stores) {
			ds := pc.dt.Backend.(*DataStack)
			nbs, hard, _ := ds.RemovePlugin(name, pc.dt.Logger)
			if hard != nil {
				rt.Errorf("Skipping removal of plugin content layer %s because of bad store errors: %v\n", name, hard)
			} else {
				pc.dt.ReplaceBackend(rt, nbs)
			}
		})
		delete(pc.AvailableProviders, name)
	}

	pc.Tracef("removePluginProvider Finished: %s\n", provider)
	return nil
}

func (pc *PluginController) UploadPlugin(c *gin.Context, fileRoot, name string) (*models.PluginProviderUploadInfo, *models.Error) {
	if err := os.MkdirAll(path.Join(fileRoot, `plugins`), 0755); err != nil {
		pc.Errorf("Unable to create plugins directory: %v", err)
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
