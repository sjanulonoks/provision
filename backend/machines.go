package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"path"
	"reflect"
	"strings"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	"github.com/pborman/uuid"
)

// Machine represents a single bare-metal system that the provisioner
// should manage the boot environment for.
// swagger:model
type Machine struct {
	*models.Machine
	validate
	// used during AfterSave() and AfterRemove() to handle boot environment changes.
	oldBootEnv string
	// used during AfterSave() and AfterRemove() to handle boot environment changes.
	oldStage string
}

func (obj *Machine) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Machine) SaveClean() store.KeySaver {
	mod := *obj.Machine
	mod.ClearValidation()
	return toBackend(&mod, obj.rt)
}

func (n *Machine) HasTask(s string) bool {
	for _, p := range n.Tasks {
		if p == s {
			return true
		}
	}
	return false
}

func (n *Machine) Indexes() map[string]index.Maker {
	fix := AsMachine
	res := index.MakeBaseIndexes(n)
	res["Uuid"] = index.Make(
		true,
		"UUID string",
		func(i, j models.Model) bool { return fix(i).Uuid.String() < fix(j).Uuid.String() },
		func(ref models.Model) (gte, gt index.Test) {
			refUuid := fix(ref).Uuid.String()
			return func(s models.Model) bool {
					return fix(s).Uuid.String() >= refUuid
				},
				func(s models.Model) bool {
					return fix(s).Uuid.String() > refUuid
				}
		},
		func(s string) (models.Model, error) {
			id := uuid.Parse(s)
			if id == nil {
				return nil, fmt.Errorf("Invalid UUID: %s", s)
			}
			m := fix(n.New())
			m.Uuid = id
			return m, nil
		})
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
			m := fix(n.New())
			m.Name = s
			return m, nil
		})
	res["BootEnv"] = index.Make(
		false,
		"string",
		func(i, j models.Model) bool { return fix(i).BootEnv < fix(j).BootEnv },
		func(ref models.Model) (gte, gt index.Test) {
			refBootEnv := fix(ref).BootEnv
			return func(s models.Model) bool {
					return fix(s).BootEnv >= refBootEnv
				},
				func(s models.Model) bool {
					return fix(s).BootEnv > refBootEnv
				}
		},
		func(s string) (models.Model, error) {
			m := fix(n.New())
			m.BootEnv = s
			return m, nil
		})
	res["Address"] = index.Make(
		false,
		"IP Address",
		func(i, j models.Model) bool {
			n, o := big.Int{}, big.Int{}
			n.SetBytes(fix(i).Address.To16())
			o.SetBytes(fix(j).Address.To16())
			return n.Cmp(&o) == -1
		},
		func(ref models.Model) (gte, gt index.Test) {
			addr := &big.Int{}
			addr.SetBytes(fix(ref).Address.To16())
			return func(s models.Model) bool {
					o := big.Int{}
					o.SetBytes(fix(s).Address.To16())
					return o.Cmp(addr) != -1
				},
				func(s models.Model) bool {
					o := big.Int{}
					o.SetBytes(fix(s).Address.To16())
					return o.Cmp(addr) == 1
				}
		},
		func(s string) (models.Model, error) {
			addr := net.ParseIP(s)
			if addr == nil {
				return nil, fmt.Errorf("Invalid address: %s", s)
			}
			m := fix(n.New())
			m.Address = addr
			return m, nil
		})
	res["Runnable"] = index.Make(
		false,
		"boolean",
		func(i, j models.Model) bool {
			return (!fix(i).Runnable) && fix(j).Runnable
		},
		func(ref models.Model) (gte, gt index.Test) {
			avail := fix(ref).Runnable
			return func(s models.Model) bool {
					v := fix(s).Runnable
					return v || (v == avail)
				},
				func(s models.Model) bool {
					return fix(s).Runnable && !avail
				}
		},
		func(s string) (models.Model, error) {
			res := fix(n.New())
			switch s {
			case "true":
				res.Runnable = true
			case "false":
				res.Runnable = false
			default:
				return nil, errors.New("Runnable must be true or false")
			}
			return res, nil
		})
	return res
}

func (n *Machine) ParameterMaker(rt *RequestTracker, parameter string) (index.Maker, error) {
	fix := AsMachine
	pobj := rt.Find("params", parameter)
	if pobj == nil {
		return index.Maker{}, fmt.Errorf("Filter not found: %s", parameter)
	}
	param := AsParam(pobj)

	return index.Make(
		false,
		"parameter",
		func(i, j models.Model) bool {
			ip, _ := rt.GetParam(fix(i), parameter, true)
			jp, _ := rt.GetParam(fix(j), parameter, true)

			// If both are nil, the Less is i < j == false
			if ip == nil && jp == nil {
				return false
			}
			// If ip is nil, the Less is i < j == true
			if ip == nil {
				if _, ok := jp.(bool); ok {
					return jp.(bool)
				}
				return true
			}
			// If jp is nil, the Less is i < j == false
			if jp == nil {
				return false
			}

			if _, ok := ip.(string); ok {
				return ip.(string) < jp.(string)
			}
			if _, ok := ip.(bool); ok {
				return jp.(bool) && !ip.(bool)
			}
			if _, ok := ip.(int); ok {
				return ip.(int) < jp.(int)
			}

			return false
		},
		func(ref models.Model) (gte, gt index.Test) {
			jp, _ := rt.GetParam(fix(ref), parameter, true)
			return func(s models.Model) bool {
					ip, _ := rt.GetParam(fix(ref), parameter, true)

					// If both are nil, the Less is i >= j == true
					if ip == nil && jp == nil {
						return true
					}
					// If ip is nil, the Less is i >= j == false
					if ip == nil {
						if _, ok := jp.(bool); ok {
							return !jp.(bool)
						}
						return false
					}
					// If jp is nil, the Less is i >= j == true
					if jp == nil {
						return true
					}

					if _, ok := ip.(string); ok {
						return ip.(string) >= jp.(string)
					}
					if _, ok := ip.(bool); ok {
						return ip.(bool) || ip.(bool) == jp.(bool)
					}
					if _, ok := ip.(int); ok {
						return ip.(int) >= jp.(int)
					}
					return false
				},
				func(s models.Model) bool {
					ip, _ := rt.GetParam(fix(s), parameter, true)

					// If both are nil, the Less is i > j == false
					if ip == nil && jp == nil {
						return false
					}
					// If ip is nil, the Less is i > j == false
					if ip == nil {
						return false
					}
					// If jp is nil, the Less is i > j == true
					if jp == nil {
						if _, ok := ip.(bool); ok {
							return ip.(bool)
						}
						return true
					}

					if _, ok := ip.(string); ok {
						return ip.(string) > jp.(string)
					}
					if _, ok := ip.(bool); ok {
						return ip.(bool) && !jp.(bool)
					}
					if _, ok := ip.(int); ok {
						return ip.(int) > jp.(int)
					}
					return false
				}
		},
		func(s string) (models.Model, error) {
			res := fix(n.New())
			res.Params = map[string]interface{}{}

			var obj interface{}
			err := json.Unmarshal([]byte(s), &obj)
			if err != nil {
				return nil, err
			}
			if err := param.ValidateValue(obj); err != nil {
				return nil, err
			}
			res.Params[parameter] = obj
			return res, nil
		}), nil

}

// HexAddress returns Address in raw hexadecimal format, suitable for
// pxelinux and elilo usage.
func (n *Machine) HexAddress() string {
	return models.Hexaddr(n.Address)
}

func (n *Machine) ShortName() string {
	idx := strings.Index(n.Name, ".")
	if idx == -1 {
		return n.Name
	}
	return n.Name[:idx]
}

func (n *Machine) Path() string {
	return path.Join(n.Prefix(), n.UUID())
}

func (n *Machine) HasProfile(name string) bool {
	for _, e := range n.Profiles {
		if e == name {
			return true
		}
	}
	return false
}

func (n *Machine) New() store.KeySaver {
	res := &Machine{Machine: &models.Machine{}}
	if n.Machine != nil && n.ChangeForced() {
		res.ForceChange()
	}
	res.Tasks = []string{}
	res.Profiles = []string{}
	res.rt = n.rt
	return res
}

func (n *Machine) OnCreate() error {
	n.oldStage = "none"
	n.oldBootEnv = "local"
	if n.Stage == "" {
		n.Stage = n.rt.dt.pref("defaultStage")
	}
	if n.BootEnv == "" {
		n.BootEnv = n.rt.dt.pref("defaultBootEnv")
	}
	if n.Tasks == nil {
		n.Tasks = []string{}
	}
	// Migrate old params to new Params
	if n.Profile.Params != nil {
		n.Params = n.Profile.Params
		n.Profile.Params = nil
	}
	// All machines start runnable.
	n.Runnable = true
	if n.Tasks != nil && len(n.Tasks) > 0 {
		n.CurrentTask = -1
	}
	n.Validate()
	return n.MakeError(422, ValidationError, n)
}

func (n *Machine) Validate() {
	if n.Uuid == nil {
		n.Errorf("Machine %#v was not assigned a uuid!", n)
	}
	n.Machine.Validate()
	validateMaybeZeroIP4(n, n.Address)
	n.AddError(index.CheckUnique(n, n.rt.stores("machines").Items()))
	n.SetValid()
	objs := n.rt.stores
	tasks := objs("tasks")
	profiles := objs("profiles")
	bootenvs := objs("bootenvs")
	stages := objs("stages")
	wantedProfiles := map[string]int{}
	for i, profileName := range n.Profiles {
		var found models.Model
		if profiles != nil {
			found = profiles.Find(profileName)
		}
		if found == nil {
			n.Errorf("Profile %s (at %d) does not exist", profileName, i)
		} else {
			if alreadyAt, ok := wantedProfiles[profileName]; ok {
				n.Errorf("Duplicate profile %s: at %d and %d", profileName, alreadyAt, i)
				n.SetInvalid() // Force Fatal
			} else {
				wantedProfiles[profileName] = i
			}
		}
	}
	if stages == nil {
		// No stages at all
		n.Errorf("Stage %s does not exist", n.Stage)
		n.SetInvalid() // Force Fatal
	} else {
		if nbFound := stages.Find(n.Stage); nbFound == nil {
			// Our particular stage is missing
			n.CurrentTask = 0
			n.Tasks = []string{}
			n.Errorf("Stage %s does not exist", n.Stage)
			n.SetInvalid() // Force Fatal
		} else {
			stage := AsStage(nbFound)
			if !stage.Available {
				// We are changing stages, but the target stage is not available
				n.CurrentTask = 0
				n.Tasks = []string{}
				n.Errorf("Machine %s wants Stage %s, which is not available", n.UUID(), n.Stage)
			} else if n.oldStage != n.Stage {
				if obFound := stages.Find(n.oldStage); obFound != nil {
					oldStage := AsStage(obFound)
					oldStage.Render(n.rt, n, n).deregister(n.rt.dt.FS)
				}
				// Only change bootenv if specified
				if stage.BootEnv != "" {
					// BootEnv should still be valid because Stage is valid.
					n.BootEnv = stage.BootEnv
					// If the bootenv changes, force the machine to not Runnable.
					// This keeps the task list from advancing in the wrong
					// BootEnv.
					if n.oldBootEnv != n.BootEnv {
						n.Runnable = false
					}
				}
				n.Tasks = make([]string, len(stage.Tasks))
				copy(n.Tasks, stage.Tasks)
				if len(n.Tasks) > 0 {
					n.CurrentTask = -1
				} else {
					n.CurrentTask = 0
				}
				stage.Render(n.rt, n, n).register(n.rt.dt.FS)
			}
		}
	}

	if bootenvs == nil {
		n.Errorf("Bootenv %s does not exist", n.BootEnv)
	} else {
		if nbFound := bootenvs.Find(n.BootEnv); nbFound == nil {
			n.Errorf("Bootenv %s does not exist", n.BootEnv)
		} else {
			env := AsBootEnv(nbFound)
			if env.OnlyUnknown {
				n.Errorf("BootEnv %s does not allow Machine assignments, it has the OnlyUnknown flag.", env.Name)
			} else if !env.Available {
				n.Errorf("BootEnv %s is not available", n.BootEnv)
			} else {
				if obFound := bootenvs.Find(n.oldBootEnv); obFound != nil {
					oldEnv := AsBootEnv(obFound)
					oldEnv.Render(n.rt, n, n).deregister(n.rt.dt.FS)
				}
				env.Render(n.rt, n, n).register(n.rt.dt.FS)
			}
		}
	}

	for i, taskName := range n.Tasks {
		if tasks == nil || tasks.Find(taskName) == nil {
			n.Errorf("Task %s (at %d) does not exist", taskName, i)
		}
	}
	n.SetAvailable()
}

func (n *Machine) BeforeSave() error {
	// Always make sure we have a secret
	if n.Secret == "" {
		n.Secret = randString(16)
	}
	if n.oldStage == "" && n.Stage != "" {
		n.oldStage = n.Stage
	}

	if n.oldBootEnv == "" && n.BootEnv != "" {
		n.oldBootEnv = n.BootEnv
	}
	n.Validate()
	if !n.Useable() {
		return n.MakeError(422, ValidationError, n)
	}
	if !n.Available {
		n.Runnable = false
	}

	// Set the features meta tag.
	n.ClearFeatures()
	env := n.rt.stores("bootenvs").Find(n.BootEnv)
	if env != nil {
		// Glean OS
		if n.oldBootEnv != n.BootEnv &&
			strings.HasSuffix(n.BootEnv, "-install") {
			n.OS = env.(*BootEnv).OS.Name
		}
		n.MergeFeatures(env.(*BootEnv).Features())
	}
	stage := n.rt.stores("stages").Find(n.Stage)
	if stage != nil {
		n.MergeFeatures(stage.(*Stage).Features())
	}
	if n.HasFeature("original-change-stage") {
		n.RemoveFeature("change-stage-v2")
	}
	if !n.HasFeature("change-stage-v2") {
		n.AddFeature("original-change-stage")
	}

	return nil
}
func (n *Machine) AfterSave() {
	n.oldStage = n.Stage
}

func (n *Machine) OnLoad() error {
	if n.Stage == "" {
		n.Stage = "none"
	}
	defer func() { n.rt = nil }()

	// This mustSave part is just to keep us from resaving all the machines on startup.
	mustSave := false
	if n.Secret == "" {
		mustSave = true
	}

	// Migrate old params to new Params
	if n.Profile.Params != nil {
		mustSave = true
		n.Params = n.Profile.Params
		n.Profile.Params = nil
	}

	err := n.BeforeSave()
	if err == nil && mustSave {
		v := n.SaveValidation()
		n.ClearValidation()
		err = n.rt.stores("machines").backingStore.Save(n.Key(), n)
		n.RestoreValidation(v)
	}
	return err
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	oldm := AsMachine(oldThing)
	n.oldBootEnv = oldm.BootEnv
	n.oldStage = oldm.Stage

	// If we are changing stages and we aren't done running tasks,
	// Fail unless the users marks a force
	// If we have a stage set, don't change bootenv unless force
	if n.Stage == "" {
		n.Stage = "none"
	}
	e := &models.Error{
		Code:  http.StatusUnprocessableEntity,
		Type:  ValidationError,
		Model: n.Prefix(),
		Key:   n.Key(),
	}
	if n.oldStage != n.Stage && oldm.CurrentTask != len(oldm.Tasks) && !n.ChangeForced() {
		e.Errorf("Can not change stages with pending tasks unless forced")
	}
	if n.oldStage == n.Stage && !(n.CurrentTask == -1 || n.CurrentTask == oldm.CurrentTask) {
		e.Errorf("Cannot change CurrentTask from %d to %d", oldm.CurrentTask, n.CurrentTask)
	}
	if n.CurrentTask != -1 && !reflect.DeepEqual(oldm.Tasks, n.Tasks) {
		runningBound := n.CurrentTask
		if runningBound != len(oldm.Tasks) {
			runningBound += 1
		}
		if len(n.Tasks) < len(oldm.Tasks) && len(n.Tasks) < runningBound {
			e.Errorf("Cannot remove tasks that have already executed or are already executing")
		} else if !reflect.DeepEqual(n.Tasks[:runningBound], oldm.Tasks[:runningBound]) {
			e.Errorf("Cannot change tasks that have already executed or are executing")
		}
	}
	if n.Stage != "none" && n.oldStage == n.Stage && n.oldBootEnv != n.BootEnv && !n.ChangeForced() {
		e.Errorf("Can not change bootenv while in a stage unless forced. old: %s new %s", n.oldBootEnv, n.BootEnv)
	}
	// Id we go from having no tasks to having tasks, set the CurrentTask to -1
	if n.Runnable && len(oldm.Tasks) == 0 && len(n.Tasks) != 0 {
		n.CurrentTask = -1
	}

	return e.HasError()
}

func (n *Machine) AfterDelete() {
	e := &models.Error{}
	if b := n.rt.stores("bootenvs").Find(n.BootEnv); b != nil {
		AsBootEnv(b).Render(n.rt, n, e).deregister(n.rt.dt.FS)
	}
	if s := n.rt.stores("stages").Find(n.Stage); s != nil {
		AsStage(s).Render(n.rt, n, e).deregister(n.rt.dt.FS)
	}
	if j := n.rt.stores("jobs").Find(n.CurrentJob.String()); j != nil {
		job := AsJob(j)
		job.Current = false
		n.rt.Save(job)
	}
}

func AsMachine(o models.Model) *Machine {
	return o.(*Machine)
}

func AsMachines(o []models.Model) []*Machine {
	res := make([]*Machine, len(o))
	for i := range o {
		res[i] = AsMachine(o[i])
	}
	return res
}

var machineLockMap = map[string][]string{
	"get":     []string{"stages", "bootenvs", "machines", "profiles", "params"},
	"create":  []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"update":  []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"patch":   []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"delete":  []string{"stages", "bootenvs", "machines", "jobs", "tasks"},
	"actions": []string{"stages", "bootenvs", "machines", "profiles", "params"},
}

func (m *Machine) Locks(action string) []string {
	return machineLockMap[action]
}
