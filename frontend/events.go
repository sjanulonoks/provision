package frontend

import (
	"net/http"

	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
)

// EventBodyParameter is used to create an Event
// swagger:parameters postEvent
type EventBodyParameter struct {
	// in: body
	Body *backend.Event
}

func (f *Frontend) InitEventApi() {
	// swagger:route POST /events Events postEvent
	//
	// Create an Event
	//
	// Create an Event from the provided object
	//
	//      Responses:
	//       204: NoContentResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/events",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "events", "post", "*") {
				return
			}
			event := backend.Event{}
			if !assureDecode(c, &event) {
				return
			}

			if err := f.pubs.PublishEvent(&event); err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}

			} else {
				c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
			}
		})
}
