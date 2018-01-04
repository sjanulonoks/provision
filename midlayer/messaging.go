package midlayer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
)

type PluginClientRequestTracker struct {
	c chan *models.PluginClientReply
	l logger.Logger
}

type PluginClient struct {
	logger.Logger
	plugin   string
	cmd      *exec.Cmd
	stderr   io.ReadCloser
	stdout   io.ReadCloser
	stdin    io.WriteCloser
	finished chan bool
	lock     sync.Mutex
	nextId   int
	pending  map[int]*PluginClientRequestTracker

	publock   sync.Mutex
	inflight  int
	unloading bool
}

func (pc *PluginClient) ReadLog() {
	// read command's stderr line by line - for logging
	in := bufio.NewScanner(pc.stderr)
	for in.Scan() {
		pc.Infof("Plugin %s: %s", pc.plugin, in.Text()) // write each line to your log, or anything you need
	}
	if err := in.Err(); err != nil {
		pc.Errorf("Plugin %s: error: %s", pc.plugin, err)
	}
	pc.finished <- true
}

func (pc *PluginClient) ReadReply() {
	// read command's stdout line by line - for replies
	in := bufio.NewScanner(pc.stdout)
	for in.Scan() {
		jsonString := in.Text()

		var resp models.PluginClientReply
		err := json.Unmarshal([]byte(jsonString), &resp)
		if err != nil {
			pc.Errorf("Failed to process: %v\n", err)
			continue
		}

		req, ok := pc.pending[resp.Id]
		if !ok {
			pc.Warnf("Failed to find request for: %v\n", resp.Id)
			continue
		}
		req.c <- &resp
		req.l.Debugf("Action processed")

		pc.lock.Lock()
		delete(pc.pending, resp.Id)
		pc.lock.Unlock()
	}
	if err := in.Err(); err != nil {
		pc.Warnf("Reply %s: error: %s", pc.plugin, err)
	}
	pc.finished <- true
}

func (pc *PluginClient) writeRequest(l logger.Logger, action string, data interface{}) (chan *models.PluginClientReply, error) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	mychan := &PluginClientRequestTracker{
		c: make(chan *models.PluginClientReply),
		l: l,
	}
	id := pc.nextId
	pc.pending[id] = mychan
	pc.nextId += 1

	req := &models.PluginClientRequest{Id: id, Action: action}

	if dataBytes, err := json.Marshal(data); err != nil {
		delete(pc.pending, id)
		l.Errorf("%s:%s: Error marshalling request data: %v", pc.plugin, action, err)
		return mychan.c, err
	} else {
		req.Data = dataBytes
	}

	if bytes, err := json.Marshal(req); err != nil {
		delete(pc.pending, id)
		l.Errorf("%s:%s Error marshalling request: %v", pc.plugin, action, err)
		return mychan.c, err
	} else {
		bytes = append(bytes, "\n"...)
		n, err := pc.stdin.Write(bytes)
		if err != nil {
			l.Errorf("%s:%s: Error sending request to plugin: %v", pc.plugin, action, err)
			return mychan.c, err
		}
		if n != len(bytes) {
			l.Errorf("%s:%s: Only sent %d out of %d bytes to plugin", pc.plugin, action, n, len(bytes))
			return mychan.c, fmt.Errorf("Failed to write all bytes: %d (%d)\n", len(bytes), n)
		}
	}
	l.Infof("%s:%s: Request sent", pc.plugin, action)
	return mychan.c, nil
}

func (pc *PluginClient) Config(params map[string]interface{}) error {
	if mychan, err := pc.writeRequest(pc.Logger, "Config", params); err != nil {
		return err
	} else {
		s := <-mychan
		if s.HasError() {
			return s.Error()
		}
	}
	return nil
}

func (pc *PluginClient) Reserve() error {
	pc.publock.Lock()
	defer pc.publock.Unlock()

	if pc.unloading {
		return fmt.Errorf("Publish not available %s: unloading\n", pc.plugin)
	}
	pc.inflight += 1
	return nil
}

func (pc *PluginClient) Release() {
	pc.publock.Lock()
	defer pc.publock.Unlock()
	pc.inflight -= 1
}

func (pc *PluginClient) Unload() {
	pc.publock.Lock()
	pc.unloading = true
	for pc.inflight != 0 {
		pc.publock.Unlock()
		time.Sleep(time.Millisecond * 15)
		pc.publock.Lock()
	}
	pc.publock.Unlock()
	return
}

func (pc *PluginClient) Publish(e *models.Event) error {
	if mychan, err := pc.writeRequest(pc.Logger, "Publish", e); err != nil {
		return err
	} else {
		s := <-mychan

		if s.HasError() {
			return s.Error()
		}
	}
	return nil
}

func (pc *PluginClient) Action(rt *backend.RequestTracker, a *models.MachineAction) error {
	if mychan, err := pc.writeRequest(rt.Logger, "Action", a); err != nil {
		return err
	} else {
		s := <-mychan
		if s.HasError() {
			return s.Error()
		}
	}
	return nil
}

func (pc *PluginClient) Stop() error {
	// Close stdin / writer.  To close, the program.
	pc.stdin.Close()

	// Wait for reader to exit
	<-pc.finished
	<-pc.finished

	// Wait for exit
	pc.cmd.Wait()
	return nil
}

func NewPluginClient(plugin string, l logger.Logger, apiPort int, path string, params map[string]interface{}) (answer *PluginClient, theErr error) {
	answer = &PluginClient{plugin: plugin, Logger: l, pending: make(map[int]*PluginClientRequestTracker, 0)}

	answer.cmd = exec.Command(path, "listen")
	// Setup env vars to run drpcli - auth should be parameters.
	env := os.Environ()
	env = append(env, fmt.Sprintf("RS_ENDPOINT=https://127.0.0.1:%d", apiPort))
	answer.cmd.Env = env

	var err2 error
	answer.stderr, err2 = answer.cmd.StderrPipe()
	if err2 != nil {
		return nil, err2
	}
	answer.stdout, err2 = answer.cmd.StdoutPipe()
	if err2 != nil {
		return nil, err2
	}
	answer.stdin, err2 = answer.cmd.StdinPipe()
	if err2 != nil {
		return nil, err2
	}

	answer.finished = make(chan bool, 2)
	go answer.ReadLog()
	go answer.ReadReply()

	answer.cmd.Start()

	terr := answer.Config(params)
	if terr != nil {
		answer.Stop()
		theErr = terr
		return
	}
	return
}
