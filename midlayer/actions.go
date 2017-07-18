package midlayer

import (
	"fmt"
	"sort"

	"github.com/digitalrebar/provision/backend"
)

// Plugins can provide actions for machines
// Assumes that there are parameters on the
// call in addition to the machine.
//
// swagger:model
type AvailableAction struct {
	Command        string
	RequiredParams []*backend.Param
	OptionalParams []*backend.Param

	plugin *RunningPlugin
}

type MachineAction struct {
	Command string
	Params  map[string]interface{}
	Machine *backend.Machine
}

type MachineActions struct {
	actions map[string]*AvailableAction
}

func NewMachineActions() *MachineActions {
	return &MachineActions{actions: make(map[string]*AvailableAction, 0)}
}

func (ma *MachineActions) Add(aa *AvailableAction) error {
	if _, ok := ma.actions[aa.Command]; ok {
		return fmt.Errorf("Duplicate Action %s: already present\n", aa.Command)
	}
	ma.actions[aa.Command] = aa
	return nil
}

func (ma *MachineActions) Remove(aa *AvailableAction) error {
	if _, ok := ma.actions[aa.Command]; !ok {
		return fmt.Errorf("Missing Action %s: already removed\n", aa.Command)
	}
	delete(ma.actions, aa.Command)
	return nil
}

func (ma *MachineActions) List() []*AvailableAction {
	// get the list of keys and sort them
	keys := []string{}
	for key := range ma.actions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	answer := []*AvailableAction{}
	for _, key := range keys {
		answer = append(answer, ma.actions[key])
	}
	return answer

}

func (ma *MachineActions) Get(name string) (a *AvailableAction, ok bool) {
	a, ok = ma.actions[name]
	return
}

func (ma *MachineActions) Run(aa *AvailableAction, maa *MachineAction) error {
	return aa.plugin.Client.Action(maa)
}
