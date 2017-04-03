package backend

import (
	"io/ioutil"
	"net"
	"path"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/pborman/uuid"
)

const (
	tmplDefault = `Machine: 
Name = {{.Machine.Name}}
HexAddress = {{.Machine.HexAddress}}
ShortName = {{.Machine.ShortName}}
FooParam = {{.Param "foo"}}

BootEnv:
Name = {{.Env.Name}}

{{if .ParamExists "fred"}}{{.Param "fred"}}{{end}}

RenderData:
ProvisionerAddress = {{.ProvisionerAddress}}
ProvisionerURL = {{.ProvisionerURL}}
CommandURL = {{.CommandURL}}
BootParams = {{.BootParams}}`
	tmplDefaultRenderedWithoutFred = `Machine: 
Name = Test Name
HexAddress = C0A87C0B
ShortName = Test Name
FooParam = bar

BootEnv:
Name = default



RenderData:
ProvisionerAddress = 127.0.0.1
ProvisionerURL = http://127.0.0.1:8091
CommandURL = CURL
BootParams = default`
	tmplDefaultRenderedWithFred = `Machine: 
Name = Test Name
HexAddress = C0A87C0B
ShortName = Test Name
FooParam = bar

BootEnv:
Name = default

fred = fred

RenderData:
ProvisionerAddress = 127.0.0.1
ProvisionerURL = http://127.0.0.1:8091
CommandURL = CURL
BootParams = default`
	tmplNothing = `Nothing`
)

func TestRenderData(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	objs := []crudTest{
		{"Create default template", dt.create, &Template{p: dt, ID: "default", Contents: tmplDefault}, true},
		{"Create nothing template", dt.create, &Template{p: dt, ID: "nothing", Contents: tmplNothing}, true},
		{"Create default bootenv", dt.create, &BootEnv{p: dt, Name: "default", Templates: []TemplateInfo{{Name: "ipxe", Path: "machines/{{.Machine.UUID}}/file", ID: "default"}}, BootParams: "{{.Env.Name}}"}, true},
		{"Create nothing bootenv", dt.create, &BootEnv{p: dt, Name: "nothing", Templates: []TemplateInfo{{Name: "ipxe", Path: "machines/{{.Machine.UUID}}/file", ID: "nothing"}}, BootParams: "{{.Env.Name}}"}, true},
	}
	for _, obj := range objs {
		obj.Test(t)
	}
	machine := dt.NewMachine()
	machine.Uuid = uuid.NewRandom()
	machine.Name = "Test Name"
	machine.Address = net.ParseIP("192.168.124.11")
	machine.BootEnv = "default"
	machine.Params = map[string]interface{}{"foo": "bar"}
	created, err := dt.create(machine)
	if !created {
		t.Errorf("Failed to create new test machine: %v", err)
		return
	} else {
		t.Logf("Created new test machine")
	}
	genLoc := path.Join(dt.FileRoot, "machines", machine.UUID(), "file")
	buf, err := ioutil.ReadFile(genLoc)
	if err != nil {
		t.Errorf("Failed to read %s: %v", genLoc, err)
	} else if string(buf) != tmplDefaultRenderedWithoutFred {
		t.Errorf("Failed to render expected template!\nExpected:\n%s\n\nGot:\n%s", tmplDefaultRenderedWithoutFred, string(buf))
	} else {
		t.Logf("BootEnv default without fred rendered properly for test machine")
	}

	machine.Params = map[string]interface{}{"foo": "bar", "fred": "fred = fred"}
	saved, err := dt.save(machine)
	if !saved {
		t.Errorf("Failed to save test machine with new bootenv: %v", err)
	}
	buf, err = ioutil.ReadFile(genLoc)
	if err != nil {
		t.Errorf("Failed to read %s: %v", genLoc, err)
	} else if string(buf) != tmplDefaultRenderedWithFred {
		t.Errorf("Failed to render expected template!\nExpected:\n%s\n\nGot:\n%s", tmplDefaultRenderedWithFred, string(buf))
	} else {
		t.Logf("BootEnv default with fred rendered properly for test machine")
	}

	machine.BootEnv = "nothing"
	saved, err = dt.save(machine)
	if !saved {
		t.Errorf("Failed to save test machine with new bootenv: %v", err)
	}
	buf, err = ioutil.ReadFile(genLoc)
	if err != nil {
		t.Errorf("Failed to read %s: %v", genLoc, err)
	} else if string(buf) != tmplNothing {
		t.Errorf("Failed to render expected template!\nExpected:\n%s\n\nGot:\n%s", tmplNothing, string(buf))
	} else {
		t.Logf("BootEnv nothing rendered properly for test machine")
	}
}
