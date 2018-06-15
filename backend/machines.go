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
type Machine struct {
	*models.Machine
	validate
	// used during AfterSave() and AfterRemove() to handle boot environment changes.
	oldBootEnv, oldStage, oldWorkflow      string
	changeStageAllowed, inCreate, inRunner bool

	toDeRegister, toRegister renderers
}

func (n *Machine) SetReadOnly(b bool) {
	n.ReadOnly = b
}

func (n *Machine) InRunner() {
	n.inRunner = true
}

func (n *Machine) AllowStageChange() {
	n.changeStageAllowed = true
}

func (n *Machine) SaveClean() store.KeySaver {
	mod := *n.Machine
	mod.ClearValidation()
	return toBackend(&mod, n.rt)
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
	res["Stage"] = index.Make(
		false,
		"string",
		func(i, j models.Model) bool { return fix(i).Stage < fix(j).Stage },
		func(ref models.Model) (gte, gt index.Test) {
			refStage := fix(ref).Stage
			return func(s models.Model) bool {
					return fix(s).Stage >= refStage
				},
				func(s models.Model) bool {
					return fix(s).Stage > refStage
				}
		},
		func(s string) (models.Model, error) {
			m := fix(n.New())
			m.Stage = s
			return m, nil
		})
	res["Workflow"] = index.Make(
		false,
		"string",
		func(i, j models.Model) bool { return fix(i).Workflow < fix(j).Workflow },
		func(ref models.Model) (gte, gt index.Test) {
			refWorkflow := fix(ref).Workflow
			return func(s models.Model) bool {
					return fix(s).Workflow >= refWorkflow
				},
				func(s models.Model) bool {
					return fix(s).Workflow > refWorkflow
				}
		},
		func(s string) (models.Model, error) {
			m := fix(n.New())
			m.Workflow = s
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
	pobj := rt.find("params", parameter)
	if pobj == nil {
		return index.Maker{}, fmt.Errorf("Filter not found: %s", parameter)
	}
	param := AsParam(pobj)

	return index.Make(
		false,
		"parameter",
		func(i, j models.Model) bool {
			ip, _ := rt.GetParam(fix(i), parameter, true, false)
			jp, _ := rt.GetParam(fix(j), parameter, true, false)

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
			jp, _ := rt.GetParam(fix(ref), parameter, true, false)
			return func(s models.Model) bool {
					ip, _ := rt.GetParam(fix(s), parameter, true, false)

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
					ip, _ := rt.GetParam(fix(s), parameter, true, false)

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
				// If type is string, then just use the value
				// we leave the json parsing so that we can test quoted strings.
				if tv, ok := param.TypeValue(); ok {
					if is, ok := tv.(string); ok && is == "string" {
						obj = s
					} else {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
			if err := param.ValidateValue(obj, nil); err != nil {
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
	res.Tasks = []string{}
	res.Profiles = []string{}
	if n != nil {
		res.rt = n.rt
		if n.Machine != nil && n.ChangeForced() {
			res.ForceChange()
		}
	}
	return res
}

func (n *Machine) OnCreate() error {
	e := &models.Error{
		Code:  422,
		Model: "machines",
		Type:  ValidationError,
		Key:   n.UUID(),
	}
	n.inCreate = true
	n.oldStage = "none"
	n.oldBootEnv = "local"
	oldm := ModelToBackend(&models.Machine{}).(*Machine)
	if n.Workflow == "" {
		n.Workflow = n.rt.dt.pref("defaultWorkflow")
	}
	if n.Stage == "" {
		n.Stage = n.rt.dt.pref("defaultStage")
	}
	if n.BootEnv == "" {
		n.BootEnv = n.rt.dt.pref("defaultBootEnv")
	}
	if n.Tasks == nil {
		n.Tasks = []string{}
	}
	realStage, realEnv := n.validateChangeWorkflow(oldm, e)
	if realStage != "" {
		n.Stage = realStage
	}
	if realEnv != "" {
		n.BootEnv = realEnv
	}
	n.validateChangeStage(oldm, e)
	n.validateChangeEnv(oldm, e)
	if e.ContainsError() {
		return e
	}
	// Migrate old params to new Params
	if n.Profile.Params != nil {
		n.Params = n.Profile.Params
		n.Profile.Params = nil
	}
	n.changeStageAllowed = true
	if n.Tasks != nil && len(n.Tasks) > 0 {
		n.CurrentTask = -1
	}
	n.Runnable = true
	n.Validate()
	return n.MakeError(422, ValidationError, n)
}

func (n *Machine) Validate() {
	if n.Uuid == nil {
		n.Errorf("Machine %#v was not assigned a uuid!", n)
	}
	n.toRegister = renderers{}
	n.toDeRegister = renderers{}
	n.Machine.Validate()
	validateMaybeZeroIP4(n, n.Address)
	n.AddError(index.CheckUnique(n, n.rt.stores("machines").Items()))
	// Validate IP address on system
	if !n.Address.IsUnspecified() {
		others, err := index.All(
			index.Sort(n.Indexes()["Address"]),
			index.Eq(n.Address.String()))(n.rt.Index("machines"))
		if err == nil {
			for _, item := range others.Items() {
				m2 := AsMachine(item)
				if m2.Key() == n.Key() {
					continue
				}
				n.Errorf("Machine %s already has IP address %s", m2.UUID(), m2.Address)
			}
		}
	}
	if pk, err := n.rt.PrivateKeyFor(n); err == nil {
		ValidateParams(n.rt, n, n.Params, pk)
	} else {
		n.Errorf("Unable to get key: %v", err)
	}
	n.SetValid()
	if n.Address != nil && !n.Address.IsUnspecified() {
		others, err := index.All(
			index.Sort(n.Indexes()["Address"]),
			index.Eq(n.Address.String()))(n.rt.Index("machines"))
		if err != nil {
			n.rt.Errorf("Error getting Address index for Machines: %v", err)
			n.Errorf("Unable to check for conflicting IP addresses: %v", err)
		} else {
			switch others.Count() {
			case 0:
			case 1:
				if others.Items()[0].Key() != n.Key() {
					n.Errorf("Machine %s already has Address %s, we cannot have it.", others.Items()[0].Key(), n.Address)
					n.Address = nil
				}
			default:
				n.Errorf("Multiple other machines have address %s, we cannot have it", n.Address)
				n.Address = nil
			}
		}
	}
	// Validate profiles
	profiles := n.rt.stores("profiles")
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
	workflows := n.rt.stores("workflows")
	// Validate workflow
	if n.Workflow != "" {
		obj := workflows.Find(n.Workflow)
		if obj == nil {
			n.Errorf("Workflow %s does not exist", n.Workflow)
		} else {
			workflow := obj.(*Workflow)
			if !workflow.Available {
				n.Errorf("Machine %s wants Workflow %s, which is not available", n.UUID(), n.Workflow)
			}
		}
	}
	stages := n.rt.stores("stages")
	// Validate stage
	if n.Stage != "" {
		obj := stages.Find(n.Stage)
		if obj == nil {
			n.Errorf("Stage %s does not exist", n.Stage)
		} else {
			stage := obj.(*Stage)
			if !stage.Available {
				n.Errorf("Machine %s wants Stage %s, which is not available", n.UUID(), n.Stage)
			}
			if obFound := stages.Find(n.oldStage); obFound != nil && n.oldStage != n.Stage {
				oldStage := AsStage(obFound)
				n.toDeRegister = append(n.toDeRegister, oldStage.render(n.rt, n, n)...)
			}
			n.toRegister = append(n.toRegister, stage.render(n.rt, n, n)...)
		}
	}
	bootenvs := n.rt.stores("bootenvs")
	// Validate bootenv
	if n.BootEnv != "" {
		obj := bootenvs.Find(n.BootEnv)
		if obj == nil {
			n.Errorf("Bootenv %s does not exist", n.BootEnv)
		} else {
			env := obj.(*BootEnv)
			if env.OnlyUnknown {
				n.Errorf("BootEnv %s does not allow Machine assignments, it has the OnlyUnknown flag.", env.Name)
			}
			if !env.Available {
				n.Errorf("BootEnv %s is not available", n.BootEnv)
			}
			if n.oldBootEnv != n.BootEnv && !n.inCreate {
				n.Runnable = false
			}
			if obFound := bootenvs.Find(n.oldBootEnv); obFound != nil {
				oldEnv := AsBootEnv(obFound)
				n.toDeRegister = append(n.toDeRegister, oldEnv.render(n.rt, n, n)...)
			}
			n.toRegister = append(n.toRegister, env.render(n.rt, n, n)...)
		}
	}
	tasks := n.rt.stores("tasks")
	// Validate task list
	for i, ent := range n.Tasks {
		parts := strings.SplitN(ent, ":", 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "stage":
				if stages.Find(parts[1]) == nil {
					n.Errorf("Stage %s (at %d) does not exist", parts[1], i)
				}
			case "bootenv":
				if bootenvs.Find(parts[1]) == nil {
					n.Errorf("BootEnv %s (at %d) does not exist", parts[1], i)
				}
			case "chroot":
			default:
				n.Errorf("%s (at %d) is malformed", ent, i)
			}
		} else {
			if tasks.Find(ent) == nil {
				n.Errorf("Task %s (at %d) does not exist", ent, i)
			}
		}
	}
	if n.CurrentTask > len(n.Tasks) {
		n.CurrentTask = len(n.Tasks)
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
	if n.oldWorkflow == "" && n.Workflow != "" {
		n.oldWorkflow = n.Workflow
	}
	n.Validate()
	if !n.Useable() {
		return n.MakeError(422, ValidationError, n)
	}
	if !n.Available {
		n.Runnable = false
	}

	// Set the features meta tag.
	// Make sure the machine defaults to change-stage-v2
	n.ClearFeatures()
	n.AddFeature("change-stage-v2")
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
	if n.Available {
		if n.toDeRegister != nil {
			n.toDeRegister.deregister(n.rt.dt.FS)
		}
		if n.toRegister != nil {
			n.toRegister.register(n.rt.dt.FS)
		}
	}
	n.toDeRegister = nil
	n.toRegister = nil
	n.oldStage = n.Stage
	n.oldBootEnv = n.BootEnv
	n.oldWorkflow = n.Workflow
	n.changeStageAllowed = false
	n.inCreate = false
	n.inRunner = false
	n.rt.dt.macAddrMux.Lock()
	for _, mac := range n.HardwareAddrs {
		n.rt.dt.macAddrMap[mac] = n.UUID()
	}
	n.rt.dt.macAddrMux.Unlock()
}

func (n *Machine) OnLoad() error {
	defer func() { n.rt = nil }()
	n.Fill()
	if n.Stage == "" {
		n.Stage = "none"
	}
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
	n.rt.dt.macAddrMux.Lock()
	for _, mac := range n.HardwareAddrs {
		n.rt.dt.macAddrMap[mac] = n.UUID()
	}
	n.rt.dt.macAddrMux.Unlock()
	return err
}

func (n *Machine) validateChangeWorkflow(oldm *Machine, e *models.Error) (newStage, newEnv string) {
	if oldm.Workflow == n.Workflow {
		return
	}
	if n.Workflow == "" {
		delete(n.Params, "change-stage/map")
		if n.Stage == oldm.Stage {
			n.Stage = ""
		}
		return
	}
	workflows := n.rt.stores("workflows")
	if workflows == nil {
		e.Errorf("Workflow %s does not exist", n.Workflow)
		return
	}
	obj := workflows.Find(n.Workflow)
	if obj == nil {
		e.Errorf("Workflow %s does not exist", n.Workflow)
		return
	}
	workflow := obj.(*Workflow)
	if !workflow.Available {
		e.Errorf("Machine %s wants Workflow %s, which is not available", n.UUID(), n.Workflow)
		return
	}
	n.CurrentTask = -1
	taskList := []string{}
	lastEnv := ""
	firstStage := true
	for _, stageName := range workflow.Stages {
		stage := n.rt.find("stages", stageName).(*Stage)
		taskList = append(taskList, "stage:"+stageName)
		if firstStage {
			newStage = stage.Name
		}
		if stage.BootEnv != "" && stage.BootEnv != lastEnv {
			if firstStage {
				newEnv = stage.BootEnv
				n.BootEnv = stage.BootEnv
			}
			taskList = append(taskList, "bootenv:"+stage.BootEnv)
			lastEnv = stage.BootEnv
		}
		taskList = append(taskList, stage.Tasks...)
		firstStage = false
	}
	n.Tasks = taskList
	return
}

func (n *Machine) validateChangeStage(oldm *Machine, e *models.Error) {
	if oldm.Stage == n.Stage {
		return
	}
	if n.Stage == "" {
		n.Stage = "none"
	}
	stages := n.rt.stores("stages")
	if stages == nil {
		e.Errorf("Stage %s does not exist", n.Stage)
		return
	}
	obj := stages.Find(n.Stage)
	if obj == nil {
		e.Errorf("Stage %s does not exist", n.Stage)
		return
	}
	stage := obj.(*Stage)
	if !stage.Available && n.Workflow == "" {
		n.CurrentTask = 0
		n.Tasks = []string{}
		e.Errorf("Machine %s wants Stage %s, which is not available", n.UUID(), n.Stage)
		return
	}
	// Only change bootenv if specified
	if stage.BootEnv != "" {
		// BootEnv should still be valid because Stage is valid.
		n.BootEnv = stage.BootEnv
	}
	if n.Workflow != "" {
		// If the Machine is being managed by a Workflow, or the Stage
		// does not have any Tasks and we are creating a Machine, then
		// changing stage does not imply changing the task list.
		return
	}
	if len(stage.Tasks) > 0 || !n.inCreate {
		n.Tasks = make([]string, len(stage.Tasks))
		copy(n.Tasks, stage.Tasks)
	}
	n.CurrentTask = -1
}

func (n *Machine) validateChangeEnv(oldm *Machine, e *models.Error) {
	if n.oldBootEnv == n.BootEnv {
		return
	}
	bootEnvs := n.rt.stores("bootenvs")
	if bootEnvs == nil {
		e.Errorf("Bootenv %s does not exist", n.BootEnv)
		return
	}
	obj := bootEnvs.Find(n.BootEnv)
	if obj == nil {
		e.Errorf("Bootenv %s does not exist", n.BootEnv)
		return
	}
	env := obj.(*BootEnv)
	if env.OnlyUnknown {
		e.Errorf("BootEnv %s does not allow Machine assignments, it has the OnlyUnknown flag.", env.Name)
		return
	}
	if !env.Available {
		e.Errorf("BootEnv %s is not available", n.BootEnv)
		return
	}
}

func (n *Machine) oldOnChange(oldm *Machine, e *models.Error) {
	// If we are changing stages and we aren't done running tasks,
	// Fail unless the users marks a force
	// If we have a stage set, don't change bootenv unless force
	if n.oldStage != n.Stage &&
		len(oldm.Tasks) != 0 &&
		oldm.CurrentTask != len(oldm.Tasks) &&
		oldm.Workflow == "" &&
		!n.ChangeForced() {
		e.Errorf("Can not change stages with pending tasks unless forced")
	}
	if n.Stage != "none" && n.oldStage == n.Stage && n.oldBootEnv != n.BootEnv && !n.ChangeForced() {
		e.Errorf("Can not change bootenv while in a stage unless forced. old: %s new %s", n.oldBootEnv, n.BootEnv)
	}
	// If we go from having no tasks to having tasks, set the CurrentTask to -1
	if n.Runnable && len(oldm.Tasks) == 0 && len(n.Tasks) != 0 {
		n.CurrentTask = -1
	}
}

func (n *Machine) resetCurrentTask(oldm *Machine, e *models.Error) {
	found := false
	target := oldm.CurrentTask
	n.rt.dt.Infof("Machine %s asked to reset CurrentTask from %d to %d", n.UUID(), target, n.CurrentTask)
	for lBound := oldm.CurrentTask; lBound > -1; lBound-- {
		if lBound >= len(n.Tasks) {
			lBound = len(n.Tasks) - 1
			continue
		}
		thing := n.Tasks[lBound]
		if !strings.HasPrefix(thing, "stage:") {
			continue
		}
		obj := n.rt.find("stages", strings.TrimPrefix(thing, "stage:"))
		if obj == nil {
			return
		}
		stage := obj.(*Stage)
		if stage.BootEnv == "" {
			continue
		}
		if stage.BootEnv == n.BootEnv {
			target = lBound
			found = true
			continue
		}
		break
	}
	if found {
		n.CurrentTask = target
	}
	n.rt.Infof("Resetting CurrentTask from %d to %d", oldm.CurrentTask, n.CurrentTask)
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	oldm := AsMachine(oldThing)
	n.oldBootEnv = oldm.BootEnv
	n.oldStage = oldm.Stage
	n.oldWorkflow = oldm.Workflow
	oldPast, _, oldFuture := oldm.SplitTasks()
	newPast, _, newFuture := n.SplitTasks()
	e := &models.Error{
		Code:  http.StatusUnprocessableEntity,
		Type:  ValidationError,
		Model: n.Prefix(),
		Key:   n.Key(),
	}
	if !n.inRunner &&
		len(oldm.Tasks) > 0 &&
		!(oldm.CurrentTask == n.CurrentTask || n.CurrentTask == -1) {
		e.Errorf("Cannot change CurrentTask from %d to %d", oldm.CurrentTask, n.CurrentTask)
		return e
	}
	newStage, newEnv := n.validateChangeWorkflow(oldm, e)
	if newStage != "" {
		n.Stage = newStage
	}
	if newEnv != "" {
		n.BootEnv = newEnv
	}
	n.validateChangeStage(oldm, e)
	n.validateChangeEnv(oldm, e)
	if n.Workflow == "" {
		n.oldOnChange(oldm, e)
	} else {
		if !n.inRunner && oldm.BootEnv != n.BootEnv && newEnv == "" {
			n.Errorf("Changing machine bootenv not allowed")
		}
		if !n.inRunner && oldm.Stage != n.Stage && newStage == "" {
			n.Errorf("Changing machine stage not allowed")
		}
	}
	if e.ContainsError() {
		return e
	}
	if (n.Workflow != "" && n.Workflow == oldm.Workflow) ||
		(n.Workflow == "" && n.Stage == oldm.Stage) {
		if oldm.CurrentTask == n.CurrentTask {
			if !reflect.DeepEqual(oldPast, newPast) {
				if len(oldPast) > len(newPast) {
					e.Errorf("Cannot remove tasks that have already executed or are already executing")
				} else {
					e.Errorf("Cannot change tasks that have already executed or are executing")
				}
			}
			if !reflect.DeepEqual(oldFuture, newFuture) {
				e.Errorf("Cannot change tasks that are past the next stage transition")
			}
		} else if !reflect.DeepEqual(n.Tasks, oldm.Tasks) &&
			len(oldm.Tasks) > 0 &&
			n.CurrentTask > -1 {
			e.Errorf("Cannot change task list and current task at the same time")
		}
	}
	if n.CurrentTask == -1 && n.Runnable {
		n.resetCurrentTask(oldm, e)
	}
	return e.HasError()
}

func (n *Machine) AfterDelete() {
	e := &models.Error{}
	if b := n.rt.stores("bootenvs").Find(n.BootEnv); b != nil {
		AsBootEnv(b).render(n.rt, n, e).deregister(n.rt.dt.FS)
	}
	if s := n.rt.stores("stages").Find(n.Stage); s != nil {
		AsStage(s).render(n.rt, n, e).deregister(n.rt.dt.FS)
	}
	if j := n.rt.stores("jobs").Find(n.CurrentJob.String()); j != nil {
		job := AsJob(j)
		job.Current = false
		n.rt.Save(job)
	}
	n.rt.dt.macAddrMux.Lock()
	for _, mac := range n.HardwareAddrs {
		if v, ok := n.rt.dt.macAddrMap[mac]; ok && v == n.UUID() {
			delete(n.rt.dt.macAddrMap, mac)
		}
	}
	n.rt.DeleteKeyFor(n)
	n.rt.dt.macAddrMux.Unlock()

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
	"get":     {"stages", "bootenvs", "machines", "profiles", "params", "workflows"},
	"create":  {"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params", "workflows"},
	"update":  {"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params", "workflows"},
	"patch":   {"stages", "bootenvs", "machines", "tasks", "profiles", "templates", "params", "workflows"},
	"delete":  {"stages", "bootenvs", "machines", "jobs", "tasks"},
	"actions": {"stages", "bootenvs", "machines", "profiles", "params"},
}

func (n *Machine) Locks(action string) []string {
	return machineLockMap[action]
}
