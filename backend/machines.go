package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"path"
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
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Uuid": index.Make(
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
				m := &Machine{}
				m.Uuid = id
				return m, nil
			}),
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
				m := &Machine{}
				m.Name = s
				return m, nil
			}),
		"BootEnv": index.Make(
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
				m := &Machine{}
				m.BootEnv = s
				return m, nil
			}),
		"Address": index.Make(
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
				m := &Machine{}
				m.Address = addr
				return m, nil
			}),
		"Runnable": index.Make(
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
				res := &Machine{}
				switch s {
				case "true":
					res.Runnable = true
				case "false":
					res.Runnable = false
				default:
					return nil, errors.New("Runnable must be true or false")
				}
				return res, nil
			}),
	}
}

func (n *Machine) ParameterMaker(d Stores, parameter string) (index.Maker, error) {
	fix := AsMachine
	pobj := d("params").Find(parameter)
	if pobj == nil {
		return index.Maker{}, fmt.Errorf("Parameter %s must be defined", parameter)
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
			res := &Machine{}
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

func (n *Machine) AuthKey() string {
	return n.Key()
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

func (n *Machine) GetParams() map[string]interface{} {
	m := n.Profile.Params
	if m == nil {
		m = map[string]interface{}{}
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

func (n *Machine) GetParam(d Stores, key string, searchProfiles bool) (interface{}, bool) {
	mm := n.GetParams()
	if v, found := mm[key]; found {
		return v, true
	}
	if searchProfiles {
		for _, e := range n.Profiles {
			if p := n.getProfile(d, e); p != nil {
				if v, ok := p.GetParam(key, false); ok {
					return v, true
				}
			}
		}
		if gp := n.getProfile(d, n.p.GlobalProfileName); gp != nil {
			if v, ok := gp.Params[key]; ok {
				return v, true
			}
		}
	}
	return nil, false
}

func (n *Machine) New() store.KeySaver {
	res := &Machine{Machine: &models.Machine{}}
	res.Tasks = []string{}
	res.Profiles = []string{}
	return res
}

func (n *Machine) setDT(p *DataTracker) {
	n.p = p
}

func (n *Machine) OnCreate() error {
	bootenvs := n.stores("bootenvs")
	if n.BootEnv == "" {
		n.BootEnv = n.p.defaultBootEnv
	}
	if bootenvs.Find(n.BootEnv) == nil {
		n.Errorf("Bootenv %s does not exist", n.BootEnv)
	} else {
		// All machines start runnable.
		n.Runnable = true
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
	validateMaybeZeroIP4(n, n.Address)
	n.AddError(index.CheckUnique(n, n.stores("machines").Items()))
	n.SetValid()
	objs := n.stores
	tasks := objs("tasks")
	profiles := objs("profiles")
	bootenvs := objs("bootenvs")
	wantedProfiles := map[string]int{}
	for i, profileName := range n.Profiles {
		found := profiles.Find(profileName)
		if found == nil {
			n.Errorf("Profile %s (at %d) does not exist", profileName, i)
		} else {
			if alreadyAt, ok := wantedProfiles[profileName]; ok {
				n.Errorf("Duplicate profile %s: at %d and %d", profileName, alreadyAt, i)
			} else {
				wantedProfiles[profileName] = i
			}
		}
	}
	for i, taskName := range n.Tasks {
		if tasks.Find(taskName) == nil {
			n.Errorf("Task %s (at %d) does not exist", taskName, i)
		}
	}

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
	n.SetAvailable()
}

func (n *Machine) BeforeSave() error {
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
	return n.BeforeSave()
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	n.oldBootEnv = AsMachine(oldThing).BootEnv
	return nil
}

func (n *Machine) AfterSave() {
	// Have we changed bootenvs.  Rebuild the task lists
	if n.oldBootEnv == n.BootEnv || !n.Available {
		return
	}
	objs := n.stores
	profiles := objs("profiles")
	bootenvs := objs("bootenvs")
	// We get tasks by aggregating
	//   1. BootEnv tasks
	//   2. Profile tasks in order.
	//   3. Global Profile tasks (if they exist)

	taskList := []string{}

	env := AsBootEnv(bootenvs.Find(n.BootEnv))
	taskList = append(taskList, env.Tasks...)

	for _, pname := range n.Profiles {
		prof := AsProfile(profiles.Find(pname))
		taskList = append(taskList, prof.Tasks...)
	}
	gprof := AsProfile(profiles.Find(n.p.GlobalProfileName))
	if gprof != nil {
		taskList = append(taskList, gprof.Tasks...)
	}

	// Reset the task list, set currentTask to 0
	n.Tasks = taskList
	n.CurrentTask = -1
	if len(taskList) == 0 {
		// Already done.
		n.CurrentTask = 0
	}

	// Reset this here to keep from looping forever.
	n.oldBootEnv = n.BootEnv

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
	"get":     []string{"machines", "profiles", "params"},
	"create":  []string{"bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"update":  []string{"bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"patch":   []string{"bootenvs", "machines", "tasks", "profiles", "templates", "params"},
	"delete":  []string{"bootenvs", "machines"},
	"actions": []string{"machines", "profiles", "params"},
}

func (m *Machine) Locks(action string) []string {
	return machineLockMap[action]
}
