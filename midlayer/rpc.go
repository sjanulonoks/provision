package midlayer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/exec"

	"github.com/digitalrebar/provision/backend"
)

type pipePair struct {
	reader io.ReadCloser
	writer io.WriteCloser
	done   chan bool
}

func NewPipePair(cmd *exec.Cmd) *pipePair {
	r, _ := cmd.StdoutPipe()
	w, _ := cmd.StdinPipe()
	return &pipePair{reader: r, writer: w, done: make(chan bool)}
}

func (this *pipePair) Read(p []byte) (int, error) {
	return this.reader.Read(p)
}

func (this *pipePair) Write(p []byte) (int, error) {
	return this.writer.Write(p)
}

func (this *pipePair) Close() error {
	this.done <- true
	return nil
}

type PluginRpcClient struct {
	cmd       *exec.Cmd
	rpcClient *rpc.Client
	stderr    io.ReadCloser
	in        *pipePair
	finished  chan bool
	logger    *log.Logger
}

func NewPluginRpcClient(plugin string, logger *log.Logger, apiPort int, path string, params map[string]interface{}) (answer *PluginRpcClient, theErr error) {
	answer = &PluginRpcClient{logger: logger}

	answer.cmd = exec.Command(path, "listen")
	env := os.Environ()
	env = append(env, fmt.Sprintf("RS_ENDPOINT=https://127.0.0.1:%d", apiPort))
	answer.cmd.Env = env
	answer.in = NewPipePair(answer.cmd)

	var err2 error
	answer.stderr, err2 = answer.cmd.StderrPipe()
	if err2 != nil {
		return nil, err2
	}

	answer.finished = make(chan bool)
	go func() {
		// read command's stdout line by line
		in := bufio.NewScanner(answer.stderr)
		for in.Scan() {
			logger.Printf("Plugin " + plugin + ": " + in.Text()) // write each line to your log, or anything you need
		}
		if err := in.Err(); err != nil {
			logger.Printf("Plugin %s: error: %s", plugin, err)
		}
		answer.finished <- true

	}()

	answer.rpcClient = jsonrpc.NewClient(answer.in)

	answer.cmd.Start()

	var err backend.Error
	answer.logger.Printf("GREG: Calling plugin.config %v\n", params)
	terr := answer.rpcClient.Call("Plugin.Config", params, &err)
	answer.logger.Printf("GREG: returned plugin.config %v %v\n", terr, err)
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
	prpc.logger.Printf("GREG: Calling plugin.publish %v\n", e)
	e2 := prpc.rpcClient.Call("Plugin.Publish", *e, &err)
	prpc.logger.Printf("GREG: returned plugin.publish %v %v\n", e2, err)
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
	prpc.logger.Printf("GREG: Calling plugin.action %v\n", a)
	e2 := prpc.rpcClient.Call("Plugin.Action", *a, &err)
	prpc.logger.Printf("GREG: returned plugin.action %v %v\n", e2, err)
	if e2 != nil {
		return e2
	}
	if err.Code != 0 || len(err.Messages) > 0 {
		return &err
	}
	return nil
}

func (prpc *PluginRpcClient) Stop() error {
	// Close stdin / writer.  To close, the program.
	prpc.logger.Printf("GREG: Stopping program by closing STDIN\n")
	prpc.in.writer.Close()

	// Wait for reader to exit
	prpc.logger.Printf("GREG: Waiting for reader to finish\n")
	<-prpc.finished

	// Wait for pipe pair to be done.
	prpc.logger.Printf("GREG: Close the rpcclient\n")
	prpc.rpcClient.Close()
	prpc.logger.Printf("GREG: Wait for rpcclient conn to close\n")
	<-prpc.in.done

	// Wait for exit
	prpc.logger.Printf("GREG: Wait for command to exit\n")
	prpc.cmd.Wait()
	return nil
}
