package backend

import (
	"net"

	"github.com/digitalrebar/provision/models"
)

const (
	ValidationError     = "ValidationError"
	TemplateRenderError = "TemplateRenderError"
	StillInUseError     = "StillInUseError"
)

type validator interface {
	setRT(*RequestTracker)
	clearRT()
}

type validate struct {
	rt *RequestTracker
}

func (v *validate) setRT(rt *RequestTracker) {
	v.rt = rt
}

func (v *validate) clearRT() {
	v.rt = nil
}

func validateIP4(e models.ErrorAdder, a net.IP) {
	if a == nil {
		e.Errorf("IP Address is nil")
	} else if !a.IsGlobalUnicast() {
		e.Errorf("%s is not a valid IP address for dr-provision", a)
	}
}

func validateMaybeZeroIP4(e models.ErrorAdder, a net.IP) {
	if len(a) != 0 && !a.IsUnspecified() {
		validateIP4(e, a)
	}
}

func validateMac(e models.ErrorAdder, mac string) {
	_, err := net.ParseMAC(mac)
	e.AddError(err)
}
