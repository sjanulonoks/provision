package backend

import (
	"fmt"
	"log"
	"time"
)

// Event represents an action in the system.
// In general, the event generates for a subject
// of the form: type.action.key
//
// swagger:model
type Event struct {
	// Time of the event.
	// swagger:strfmt date-time
	Time time.Time

	// Type - object type
	Type string

	// Action - what happened
	Action string

	// Key - the id of the object
	Key string

	// Object - the data of the object.
	Object interface{}
}

type Publisher interface {
	Publish(event *Event) error
}

type Publishers struct {
	pubs   []Publisher
	logger *log.Logger
}

func NewPublishers(logger *log.Logger) *Publishers {
	return &Publishers{logger: logger, pubs: make([]Publisher, 0, 0)}
}

func (p *Publishers) Add(pp Publisher) {
	p.pubs = append(p.pubs, pp)
}

func (p *Publishers) Remove(pp Publisher) {
	for i, ppp := range p.pubs {
		if ppp == pp {
			p.pubs = append(p.pubs[:i], p.pubs[i+1:]...)
			break
		}
	}
}

func (p *Publishers) List() []Publisher {
	return p.pubs
}

func (p *Publishers) Publish(t, a, k string, o interface{}) error {
	e := &Event{Time: time.Now(), Type: t, Action: a, Key: k, Object: o}

	for _, pub := range p.pubs {
		if err := pub.Publish(e); err != nil {
			p.logger.Printf("Failed to Publish event on %#v: %#v\n", pub, err)
		}
	}

	return nil
}

func (e *Event) Text() string {
	return fmt.Sprintf("%d: %s %s %s\n", e.Time.Unix(), e.Type, e.Action, e.Key)
}
