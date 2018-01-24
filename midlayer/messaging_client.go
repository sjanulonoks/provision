package midlayer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
)

func (pc *PluginClient) post(l logger.Logger, path string, indata interface{}) ([]byte, error) {
	l.Tracef("post: started: %s, %v\n", path, indata)
	if data, err := json.Marshal(indata); err != nil {
		l.Tracef("post: error: marshal %v\n", err)
		return nil, err
	} else {
		resp, err := pc.client.Post(
			fmt.Sprintf("http://unix/api-plugin/v3%s", path),
			"application/json",
			strings.NewReader(string(data)))
		if err != nil {
			l.Tracef("post: error: call %v\n", err)
			return nil, err
		}
		defer resp.Body.Close()

		b, e := ioutil.ReadAll(resp.Body)
		if e != nil {
			l.Tracef("post: error: %v, %v\n", b, e)
			return nil, e
		}

		if resp.StatusCode >= 400 {
			berr := models.Error{}
			err := json.Unmarshal(b, &berr)
			if err != nil {
				l.Tracef("post: unmarshal error: %v, %v\n", b, e)
				return nil, e
			}
			return nil, &berr
		}

		return b, nil
	}
}

func (pc *PluginClient) get(l logger.Logger, path string) ([]byte, error) {
	l.Tracef("get: started: %s\n", path)
	uri := fmt.Sprintf("http://unix/api-plugin/v3%s", path)
	if resp, err := pc.client.Get(uri); err != nil {
		l.Tracef("get: finished: call %v\n", err)
		return nil, err
	} else {
		defer resp.Body.Close()
		b, e := ioutil.ReadAll(resp.Body)
		l.Tracef("get: finished: %v, %v\n", b, err)
		return b, e
	}
}

func (pc *PluginClient) Stop() error {
	pc.Tracef("Stop: started\n")
	// Send stop message
	_, err := pc.post(pc, "/stop", nil)
	if err != nil {
		pc.Errorf("Stop failed: %v\n", err)
	} else {
		pc.Tracef("Stop: post complete\n")
	}

	// Wait for log reader to exit
	if se, err := pc.cmd.StderrPipe(); err == nil {
		se.Close()
	}
	if so, err := pc.cmd.StdoutPipe(); err == nil {
		so.Close()
	}

	pc.Tracef("Stop: waiting for readers to stop\n")
	count := 0
	for atomic.LoadInt64(&pc.done) > 0 && count < 60 {
		pc.Tracef("Stop: waiting for readers to stop: %d\n", count)
		count += 1
		time.Sleep(1 * time.Second)
	}

	// Wait for exit
	pc.Tracef("Stop: waiting for command exit\n")
	pc.cmd.Wait()

	// Make sure that the sockets are removed
	retSocketPath := fmt.Sprintf("%s/%s.fromPlugin", pc.pc.pluginCommDir, pc.plugin)
	socketPath := fmt.Sprintf("%s/%s.toPlugin", pc.pc.pluginCommDir, pc.plugin)
	os.Remove(retSocketPath)
	os.Remove(socketPath)

	pc.Tracef("Stop: finished\n")
	return nil
}

func (pc *PluginClient) Config(params map[string]interface{}) error {
	pc.Tracef("Config %s: started\n", pc.plugin)
	_, err := pc.post(pc, "/config", params)
	pc.Tracef("Config %s: finished: %v\n", pc.plugin, err)
	return err
}

func (pc *PluginClient) Publish(e *models.Event) error {
	l := pc.NoPublish()
	l.Tracef("Publish %s: started\n", pc.plugin)
	_, err := pc.post(l.NoPublish(), "/publish", e)
	l.Tracef("Publish %s: finished: %v\n", pc.plugin, err)
	return err
}

func (pc *PluginClient) Action(rt *backend.RequestTracker, a *models.Action) (interface{}, error) {
	pc.Tracef("Action: started\n")
	bytes, err := pc.post(pc, "/action", a)
	var val interface{}
	if err == nil {
		err = json.Unmarshal(bytes, &val)
	}
	pc.Tracef("Action: finished: %v, %v\n", val, err)
	return val, err
}
