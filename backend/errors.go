package backend

import (
	"fmt"
	"net"
	"strings"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

const (
	ValidationError     = "ValidationError"
	TemplateRenderError = "TemplateRenderError"
	StillInUseError     = "StillInUseError"
)

func validateIP4(e *Error, a net.IP) {
	if a == nil {
		e.Errorf("IP Address is nil")
	} else if !a.IsGlobalUnicast() {
		e.Errorf("%s is not a valid IP address for rocketskates", a)
	}
}

func validateMaybeZeroIP4(e *Error, a net.IP) {
	if len(a) != 0 && !a.IsUnspecified() {
		validateIP4(e, a)
	}
}

func validateMac(e *Error, mac string) {
	if _, err := net.ParseMAC(mac); err != nil {
		e.Errorf(err.Error())
	}
}

// Error is the common Error type we should return for any errors.
// swagger:model
type Error struct {
	o     store.KeySaver
	Model string
	Key   string
	Type  string
	// Messages are any additional messages related to this Error
	Messages []string
	// code is the HTTP status code that should be used for this Error
	Code          int `json:"-"`
	containsError bool
}

func NewError(t string, code int, m string) *Error {
	return &Error{Type: t, Code: code, Messages: []string{m}}
}

func (e *Error) Errorf(s string, args ...interface{}) {
	e.containsError = true
	if e.o != nil {
		e.Model = e.o.Prefix()
		e.Key = e.o.Key()
	}
	if e.Messages == nil {
		e.Messages = []string{}
	}
	e.Messages = append(e.Messages, fmt.Sprintf(s, args...))
}

func (e *Error) Error() string {
	var res string
	if e.Key != "" {
		res = fmt.Sprintf("%s/%s: %s\n", e.Model, e.Key, e.Type)
	} else if e.Model != "" {
		res = fmt.Sprintf("%s: %s\n", e.Key, e.Type)
	} else {
		res = fmt.Sprintf("%s:/n", e.Type)
	}
	allMsgs := strings.Join(e.Messages, "\n")
	return res + allMsgs
}

func (e *Error) ContainsError() bool {
	return e.containsError
}

func (e *Error) Merge(src error) {
	if src == nil {
		return
	}
	if e.Messages == nil {
		e.Messages = []string{}
	}
	other, ok := src.(*Error)
	if !ok {
		e.containsError = true
		e.Messages = append(e.Messages, src.Error())
	} else if other.Messages != nil {
		e.containsError = true
		e.Messages = append(e.Messages, other.Messages...)
	}
}

func (e *Error) OrNil() error {
	if e.containsError {
		return e
	}
	return nil
}
