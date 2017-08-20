package midlayer

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/digitalrebar/provision/models"
)

type AvailableAction struct {
	models.AvailableAction

	plugin *RunningPlugin
	ma     *MachineActions

	lock      sync.Mutex
	inflight  int
	unloading bool
}

type MachineActions struct {
	actions map[string]*AvailableAction
	lock    sync.Mutex
}

func NewMachineActions() *MachineActions {
	return &MachineActions{actions: make(map[string]*AvailableAction, 0)}
}

func (ma *MachineActions) Add(model_aa *models.AvailableAction, plugin *RunningPlugin) error {
	aa := &AvailableAction{}
	aa.AvailableAction = *model_aa
	aa.plugin = plugin

	ma.lock.Lock()
	defer ma.lock.Unlock()

	if _, ok := ma.actions[aa.Command]; ok {
		return fmt.Errorf("Duplicate Action %s: already present\n", aa.Command)
	}
	ma.actions[aa.Command] = aa
	aa.ma = ma
	return nil
}

func (ma *MachineActions) Remove(aa *models.AvailableAction) error {
	var err error
	var the_aa *AvailableAction
	ma.lock.Lock()
	if ta, ok := ma.actions[aa.Command]; !ok {
		err = fmt.Errorf("Missing Action %s: already removed\n", aa.Command)
	} else {
		the_aa = ta
		delete(ma.actions, aa.Command)
	}
	ma.lock.Unlock()

	if the_aa != nil {
		the_aa.Unload()
	}

	return err
}

func (ma *MachineActions) List() []*models.AvailableAction {
	ma.lock.Lock()
	defer ma.lock.Unlock()

	// get the list of keys and sort them
	keys := []string{}
	for key := range ma.actions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	answer := []*models.AvailableAction{}
	for _, key := range keys {
		answer = append(answer, &ma.actions[key].AvailableAction)
	}
	return answer

}

func (ma *MachineActions) Get(name string) (*models.AvailableAction, bool) {
	ma.lock.Lock()
	defer ma.lock.Unlock()

	if ta, ok := ma.actions[name]; ok {
		return &ta.AvailableAction, true
	}
	return nil, false
}

func (ma *MachineActions) Run(maa *models.MachineAction) error {
	var aa *AvailableAction
	var ok bool
	ma.lock.Lock()
	aa, ok = ma.actions[maa.Command]
	if !ok {
		return fmt.Errorf("Action no longer available: %s", aa.Command)
	}
	ma.lock.Unlock()

	if err := aa.Reserve(); err != nil {
		return nil
	}
	defer aa.Release()

	return aa.plugin.Client.Action(maa)
}

func (aa *AvailableAction) Reserve() error {
	aa.lock.Lock()
	defer aa.lock.Unlock()

	if aa.unloading {
		return fmt.Errorf("Action not available %s: unloading", aa.Command)
	}
	aa.inflight += 1
	return nil
}

func (aa *AvailableAction) Release() {
	aa.lock.Lock()
	defer aa.lock.Unlock()

	aa.inflight -= 1
}

func (aa *AvailableAction) Unload() {
	aa.lock.Lock()
	aa.unloading = true
	for aa.inflight != 0 {
		aa.lock.Unlock()
		time.Sleep(time.Millisecond * 15)
		aa.lock.Lock()
	}
	aa.lock.Unlock()
	return
}
