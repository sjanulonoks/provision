package midlayer

import (
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os/exec"

	"github.com/digitalrebar/provision/backend"
)

type pipePair struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (this *pipePair) Read(p []byte) (int, error) {
	return this.reader.Read(p)
}

func (this *pipePair) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *pipePair) Close() error {
	this.writer.Close()
	return this.reader.Close()
}

type PluginRpcClient struct {
	cmd       *exec.Cmd
	rpcClient *rpc.Client
}

func NewPluginRpcClient(path string, params map[string]interface{}) (answer *PluginRpcClient, theErr error) {
	answer = &PluginRpcClient{}

	answer.cmd = exec.Command(path, "listen")
	in := pipePair{}
	in.reader, _ = answer.cmd.StdoutPipe()
	in.writer, _ = answer.cmd.StdinPipe()

	answer.rpcClient = jsonrpc.NewClient(&in)

	answer.cmd.Start()

	var err backend.Error
	terr := answer.rpcClient.Call("Plugin.Config", params, &err)
	if terr != nil {
		answer.Stop()
		theErr = terr
		return
	}
	if err.Code != 0 || len(err.Messages) > 0 {
		answer.Stop()
		theErr = &err
		return
	}

	return
}

func (prpc *PluginRpcClient) Publish(e *backend.Event) error {
	var err backend.Error
	e2 := prpc.rpcClient.Call("Plugin.Publish", *e, &err)
	if e2 != nil {
		return e2
	}
	if err.Code != 0 || len(err.Messages) > 0 {
		return &err
	}
	return nil
}

func (prpc *PluginRpcClient) Action(a *MachineAction) error {
	var err backend.Error
	e2 := prpc.rpcClient.Call("Plugin.Action", *a, &err)
	if e2 != nil {
		return e2
	}
	if err.Code != 0 || len(err.Messages) > 0 {
		return &err
	}
	return nil
}

func (prpc *PluginRpcClient) Stop() error {
	var err backend.Error
	i := 0
	e2 := prpc.rpcClient.Call("Plugin.Stop", i, &err)
	if e2 == nil {
		// Wait for exit
		prpc.cmd.Wait()
	}
	return e2
}
