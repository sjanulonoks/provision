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
	p *DataTracker
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
	return toBackend(obj.p, nil, &mod)
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

func (n *Machine) ParameterMaker(d Stores, parameter string) (index.Maker, error) {
	fix := AsMachine
	pobj := d("params").Find(parameter)
	if pobj == nil {
		return index.Maker{}, fmt.Errorf("Filter not found: %s", parameter)
	}
	param := AsParam(pobj)

	return index.Make(
		false,
		"parameter",
		func(i, j models.Model) bool {
			ip, _ := fix(i).GetParam(d, parameter, true)
			jp, _ := fix(j).GetParam(d, parameter, true)

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
			jp, _ := fix(ref).GetParam(d, parameter, true)
			return func(s models.Model) bool {
					ip, _ := fix(s).GetParam(d, parameter, true)

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
					ip, _ := fix(s).GetParam(d, parameter, true)

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
			res.Profile = models.Profile{Params: map[string]interface{}{}}

			var obj interface{}
			err := json.Unmarshal([]byte(s), &obj)
			if err != nil {
				return nil, err
			}
			if err := param.ValidateValue(obj); err != nil {
				return nil, err
			}
			res.Profile.Params[parameter] = obj
			return res, nil
		}), nil

}

func (n *Machine) Backend() store.Store {
	return n.p.getBackend(n)
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

func (n *Machine) getProfile(d Stores, key string) *Profile {
	p := d("profiles").Find(key)
	if p != nil {
		return AsProfile(p)
	}
	return nil
}

func (n *Machine) getStage(d Stores) *Stage {
	if n.Stage == "" {
		return nil
	}
	s := d("stages").Find(n.Stage)
	if s != nil {
		return AsStage(s)
	}
	return nil
}

func (n *Machine) GetParams(d Stores, aggregate bool) map[string]interface{} {
	m := map[string]interface{}{}
	if aggregate {
		// Check the global profile.
		if gp := n.getProfile(d, n.p.GlobalProfileName); gp != nil && gp.Params != nil {
			for k, v := range gp.Params {
				m[k] = v
			}
		}
		// Check the stage's profiles if it exists
		stage := n.getStage(d)
		if stage != nil {
			for _, pn := range stage.Profiles {
				if p := n.getProfile(d, pn); p != nil && p.Params != nil {
					for k, v := range p.Params {
						m[k] = v
					}
				}
			}
		}
		// Check profiles for params
		for _, e := range n.Profiles {
			if p := n.getProfile(d, e); p != nil && p.Params != nil {
				for k, v := range p.Params {
					m[k] = v
				}
			}
		}
		// The machine's Params
		if n.Profile.Params != nil {
			for k, v := range n.Profile.Params {
				m[k] = v
			}
		}
	} else {
		if n.Profile.Params != nil {
			m = n.Profile.Params
		}
	}
	return m
}

func (n *Machine) SetParams(d Stores, values map[string]interface{}) error {
	n.Profile.Params = values
	e := &models.Error{Code: 422, Type: ValidationError, Object: n}
	_, e2 := n.p.Save(d, n)
	e.AddError(e2)
	return e.HasError()
}

func (n *Machine) GetParam(d Stores, key string, aggregate bool) (interface{}, bool) {
	mm := n.GetParams(d, false)
	if v, found := mm[key]; found {
		return v, true
	}
	if aggregate {
		// Check profiles for params
		for _, e := range n.Profiles {
			if p := n.getProfile(d, e); p != nil {
				if v, ok := p.GetParam(d, key, false); ok {
					return v, true
				}
			}
		}
		// Check the stage's profiles if it exists
		stage := n.getStage(d)
		if stage != nil {
			for _, pn := range stage.Profiles {
				if p := n.getProfile(d, pn); p != nil {
					if v, ok := p.GetParam(d, key, false); ok {
						return v, true
					}
				}
			}
		}
		// Check the global profile.
		if gp := n.getProfile(d, n.p.GlobalProfileName); gp != nil {
			if v, ok := gp.Params[key]; ok {
				return v, true
			}
		}
	}
	return nil, false
}

func (n *Machine) SetParam(d Stores, key string, val interface{}) error {
	n.Profile.Params[key] = val
	e := &models.Error{Code: 422, Type: ValidationError, Object: n}
	_, e2 := n.p.Save(d, n)
	e.AddError(e2)
	return e.HasError()
}

func (n *Machine) New() store.KeySaver {
	res := &Machine{Machine: &models.Machine{}}
	if n.Machine != nil && n.ChangeForced() {
		res.ForceChange()
	}
	res.Tasks = []string{}
	res.Profiles = []string{}
	res.p = n.p
	return res
}

func (n *Machine) setDT(p *DataTracker) {
	n.p = p
}

func (n *Machine) OnCreate() error {
	if n.Stage == "" {
		n.Stage = n.p.pref("defaultStage")
		n.oldStage = "none"
	}
	if n.BootEnv == "" {
		n.BootEnv = n.p.pref("defaultBootEnv")
	}
	bootenvs := n.stores("bootenvs")
	stages := n.stores("stages")
	if n.Tasks == nil {
		n.Tasks = []string{}
	}
	if bootenvs.Find(n.BootEnv) == nil {
		n.Errorf("Bootenv %s does not exist", n.BootEnv)
	} else if stages.Find(n.Stage) == nil {
		n.Errorf("Stage %s does not exist", n.Stage)
	} else {
		// All machines start runnable.
		n.Runnable = true
	}
	if n.Tasks != nil && len(n.Tasks) > 0 {
		n.CurrentTask = -1
	}
	return n.MakeError(422, ValidationError, n)
}

func (n *Machine) Validate() {
	if n.Uuid == nil {
		n.Errorf("Machine %#v was not assigned a uuid!", n)
	}
	if n.Name == "" {
		n.Errorf("Machine %s must have a name", n.Uuid)
	}
	if strings.Contains(n.Name, "/") || strings.Contains(n.Name, "\\") {
		n.Errorf("Name must not contain a '/' or '\\'")
	}
	validateMaybeZeroIP4(n, n.Address)
	n.AddError(index.CheckUnique(n, n.stores("machines").Items()))
	n.SetValid()
	objs := n.stores
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
	for i, taskName := range n.Tasks {
		if tasks == nil || tasks.Find(taskName) == nil {
			n.Errorf("Task %s (at %d) does not exist", taskName, i)
		}
	}
	if stages == nil {
		n.Errorf("Stage %s does not exist", n.Stage)
		n.SetInvalid() // Force Fatal
	} else {
		if nbFound := stages.Find(n.Stage); nbFound == nil {
			n.CurrentTask = 0
			n.Tasks = []string{}
			n.Errorf("Stage %s does not exist", n.Stage)
			n.SetInvalid() // Force Fatal
		} else {
			stage := AsStage(nbFound)
			if !stage.Available {
				n.CurrentTask = 0
				n.Tasks = []string{}
				n.Errorf("Machine %s wants Stage %s, which is not available", n.UUID(), n.Stage)
			} else {
				// Only change bootenv if specified
				if stage.BootEnv != "" {
					// BootEnv should still be valid because Stage is valid.
					n.BootEnv = stage.BootEnv
				}

				// XXX: For sanity, check the path of templates to make sure not overlap
				// with the bootenv.  This is hard - do this last

				if obFound := stages.Find(n.oldStage); obFound != nil {
					oldStage := AsStage(obFound)
					oldStage.Render(objs, n, n).deregister(n.p.FS)
				}
				stage.Render(objs, n, n).register(n.p.FS)
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
				n.Errorf("Machine %s wants BootEnv %s, which is not available", n.UUID(), n.BootEnv)
			} else {
				if obFound := bootenvs.Find(n.oldBootEnv); obFound != nil {
					oldEnv := AsBootEnv(obFound)
					oldEnv.Render(objs, n, n).deregister(n.p.FS)
				}
				env.Render(objs, n, n).register(n.p.FS)
			}
		}
	}
	n.SetAvailable()
}

func (n *Machine) BeforeSave() error {
	// Always make sure we have a secret
	if n.Secret == "" {
		n.Secret = randString(16)
	}
	n.Validate()
	if !n.Useable() {
		return n.MakeError(422, ValidationError, n)
	}
	if !n.Available {
		n.Runnable = false
	}
	return nil
}

func (n *Machine) OnLoad() error {
	n.stores = func(ref string) *Store {
		return n.p.objs[ref]
	}
	if n.Stage == "" {
		n.Stage = "none"
		n.oldStage = "none"
	}
	defer func() { n.stores = nil }()

	err := n.BeforeSave()
	if err == nil {
		err = n.stores("machines").backingStore.Save(n.Key(), n)
	}
	return err
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	oldm := AsMachine(oldThing)
	n.oldBootEnv = AsMachine(oldThing).BootEnv
	n.oldStage = AsMachine(oldThing).Stage

	// If we are changing stages and we aren't done running tasks,
	// Fail unless the users marks a force
	// If we have a stage set, don't change bootenv unless force
	if n.Stage == "" {
		n.Stage = "none"
	}
	e := &models.Error{Code: http.StatusUnprocessableEntity, Type: ValidationError}
	if n.oldStage != n.Stage && oldm.CurrentTask != len(oldm.Tasks) && !n.ChangeForced() {
		e.Errorf("Can not change stages with pending tasks unless forced")
		return e
	}
	if oldm.Tasks != nil && len(oldm.Tasks) != 0 && n.Stage == oldm.Stage {
		if n.Tasks == nil || len(n.Tasks) < len(oldm.Tasks) {
			e.Errorf("Cannot remove tasks from machines without changing stage")
		}
		if !reflect.DeepEqual(n.Tasks[:len(oldm.Tasks)], oldm.Tasks) {
			e.Errorf("Can only append tasks to the task list on a machine.")
		}
	}
	if n.Stage != "none" && n.oldStage == n.Stage && n.oldBootEnv != n.BootEnv && !n.ChangeForced() {
		e.Errorf("Can not change bootenv while in a stage unless forced. old: %s new %s", n.oldBootEnv, n.BootEnv)
		return e
	}
	return nil
}

func (n *Machine) AfterSave() {
	// Have we changed stages.  Rebuild the task lists
	if n.oldStage == n.Stage || !n.Available {
		// If we don't have a stage, init structs
		if n.Stage == "" && n.Tasks == nil {
			n.Tasks = []string{}
			n.CurrentTask = -1
			_, e2 := n.p.Save(n.stores, n)
			if e2 != nil {
				n.p.Logger.Printf("Failed to save machine in after Save. %v\n", n)
			}
		}
		return
	}
	objs := n.stores
	stages := objs("stages")

	taskList := []string{}
	if obj := stages.Find(n.Stage); obj != nil {
		stage := AsStage(obj)
		taskList = append(taskList, stage.Tasks...)
	}

	// Reset the task list, set currentTask to 0
	n.Tasks = taskList
	n.CurrentTask = -1
	if len(taskList) == 0 {
		// Already done.
		n.CurrentTask = 0
	}

	// Reset this here to keep from looping forever.
	n.oldStage = n.Stage

	_, e2 := n.p.Save(objs, n)
	if e2 != nil {
		n.p.Logger.Printf("Failed to save machine in after Save. %v\n", n)
	}
}

func (n *Machine) AfterDelete() {
	e := &models.Error{}
	if b := n.stores("bootenvs").Find(n.BootEnv); b != nil {
		AsBootEnv(b).Render(n.stores, n, e).deregister(n.p.FS)
	}
	if s := n.stores("stages").Find(n.Stage); s != nil {
		AsStage(s).Render(n.stores, n, e).deregister(n.p.FS)
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
	"get":     []string{"stages", "machines", "profiles", "params"},
	"create":  []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"update":  []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"patch":   []string{"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"delete":  []string{"stages", "bootenvs", "machines"},
	"actions": []string{"stages", "machines", "profiles", "params"},
}

func (m *Machine) Locks(action string) []string {
	return machineLockMap[action]
}
