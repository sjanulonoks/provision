package frontend

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"

	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

var wsLock = &sync.Mutex{}

func (fe *Frontend) InitWebSocket() {
	fe.melody = melody.New()

	fe.ApiGroup.GET("/ws", func(c *gin.Context) {
		claim, _ := c.Get("DRP-CLAIM")
		keys := map[string]interface{}{
			"DRP-CLAIM": claim,
		}
		fe.melody.HandleRequestWithKeys(c.Writer, c.Request, keys)
	})

	fe.melody.HandleMessage(fe.websocketHandler)
}

// Callers register or deregister values.
// type.action.key = with * as wildcard spots.

func (f *Frontend) filterFunction(emap []string, claim interface{}, e *models.Event) bool {
	// Check for an event to map.
	matched := false
	for _, test := range emap {
		arr := strings.SplitN(test, ".", 3)

		if arr[0] != "*" && arr[0] != e.Type {
			continue
		}
		if arr[1] != "*" && arr[1] != e.Action {
			continue
		}
		if arr[2] != "*" && arr[2] != e.Key {
			continue
		}
		matched = true
		break
	}

	// Make sure we are authorized to see this event.
	if matched {
		matched = f.assureAuthWithClaim(nil, claim, e.Type, e.Action, e.Key)
	}
	return matched
}

func (f *Frontend) Publish(e *models.Event) error {
	if msg, err := json.Marshal(e); err != nil {
		return err
	} else {
		return f.melody.BroadcastFilter(msg, func(s *melody.Session) bool {
			var emap []string
			var claim interface{}
			hasMap := func() bool {
				wsLock.Lock()
				defer wsLock.Unlock()
				claim = s.MustGet("DRP-CLAIM")
				if val, ok := s.Get("EventMap"); !ok {
					return false
				} else {
					tmap := val.([]string)
					emap = make([]string, len(tmap), len(tmap))
					for i, v := range tmap {
						emap[i] = v
					}
					return true
				}
			}()
			if !hasMap {
				return false
			}
			return f.filterFunction(emap, claim, e)
		})
	}
}

// This never gets unloaded.
func (f *Frontend) Reserve() error {
	return nil
}
func (f *Frontend) Release() {}
func (f *Frontend) Unload()  {}

func (f *Frontend) websocketHandler(s *melody.Session, buf []byte) {
	splitMsg := bytes.SplitN(bytes.TrimSpace(buf), []byte(" "), 2)
	if len(splitMsg) != 2 {
		f.Logger.Warnf("WS: Unknown: Received message: %s\n", string(buf))
		return
	}
	prefix, msg := string(splitMsg[0]), string(splitMsg[1])
	if !(prefix == "register" || prefix == "deregister") {
		f.Logger.Warnf("WS: Invalid msg prefix %s", prefix)
		return
	}
	wsLock.Lock()
	defer wsLock.Unlock()
	val, ok := s.Get("EventMap")
	if !ok {
		val = []string{}
	}
	emap := val.([]string)
	switch prefix {
	case "register":
		for _, test := range emap {
			if test == msg {
				return
			}
		}
		emap = append(emap, msg)
	case "deregister":
		res := make([]string, 0, len(emap))
		for _, test := range emap {
			if test == msg {
				continue
			}
		}
		emap = res
	}
	s.Set("EventMap", emap)
}
