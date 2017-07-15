package frontend

import (
	"encoding/json"
	"strings"

	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
)

func (fe *Frontend) InitWebSocket() {
	fe.melody = melody.New()

	fe.ApiGroup.GET("/ws", func(c *gin.Context) {
		fe.melody.HandleRequest(c.Writer, c.Request)
	})

	fe.melody.HandleMessage(fe.websocketHandler)
}

// Callers register or deregister values.
// type.action.key = with * as wildcard spots.

func filterFunction(s *melody.Session, e *backend.Event) bool {
	val, ok := s.Get("EventMap")
	if !ok {
		return false
	}
	emap := val.([]string)
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
		return true
	}
	return false
}

func (f *Frontend) Publish(e *backend.Event) error {
	if msg, err := json.Marshal(e); err != nil {
		return err
	} else {
		f.Logger.Printf("GREG: sending message: %s\n", string(msg))
		return f.melody.BroadcastFilter(msg, func(s *melody.Session) bool {
			return filterFunction(s, e)
		})
	}
}

func (f *Frontend) websocketHandler(s *melody.Session, msg []byte) {
	str := strings.TrimSpace(string(msg))
	if strings.HasPrefix(str, "register ") {
		val, ok := s.Get("EventMap")
		if !ok {
			val = []string{}
		}
		emap := val.([]string)
		str = strings.TrimPrefix(str, "register ")
		for _, test := range emap {
			if test == str {
				return
			}
		}
		emap = append(emap, str)
		s.Set("EventMap", emap)
	} else if strings.HasPrefix(str, "deregister ") {
		val, ok := s.Get("EventMap")
		if !ok {
			return
		}
		str = strings.TrimPrefix(str, "deregister ")
		emap := val.([]string)
		newmap := []string{}
		for _, test := range emap {
			if test == str {
				continue
			}
			newmap = append(newmap, test)
		}
		s.Set("EventMap", emap)
	} else {
		f.Logger.Printf("Unknown: Received message: %s\n", str)
	}
}
