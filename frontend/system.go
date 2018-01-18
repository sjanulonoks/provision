package frontend

import (
	"github.com/digitalrebar/provision/backend"
)

// SystemActionsPathParameter used to find a System / Actions in the path
// swagger:parameters getSystemActions
type SystemActionsPathParameter struct {
	// in: query
	Plugin string `json:"plugin"`
}

// SystemActionPathParameter used to find a System / Action in the path
// swagger:parameters getSystemAction
type SystemActionPathParameter struct {
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
}

// SystemActionBodyParameter used to post a System / Action in the path
// swagger:parameters postSystemAction
type SystemActionBodyParameter struct {
	// in: path
	// required: true
	Cmd string `json:"cmd"`
	// in: query
	Plugin string `json:"plugin"`
	// in: body
	// required: true
	Body map[string]interface{}
}

func (f *Frontend) InitSystemApi() {
	profile := &backend.Profile{}
	pActions, pAction, pRun := f.makeActionEndpoints("system", profile, "name")

	// swagger:route GET /system/actions System getSystemActions
	//
	// List system actions System
	//
	// List System actions
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionsResponse
	//       401: NoSystemResponse
	//       403: NoSystemResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/system/actions", pActions)

	// swagger:route GET /system/actions/{cmd} System getSystemAction
	//
	// List specific action for System
	//
	// List specific {cmd} action for System
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//     Responses:
	//       200: ActionResponse
	//       400: ErrorResponse
	//       401: NoSystemResponse
	//       403: NoSystemResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/system/actions/:cmd", pAction)

	// swagger:route POST /system/actions/{cmd} System postSystemAction
	//
	// Call an action on the system.
	//
	// Optionally, a query parameter can be used to limit the scope to a specific plugin.
	//   e.g. ?plugin=fred
	//
	//
	//     Responses:
	//       400: ErrorResponse
	//       200: ActionPostResponse
	//       401: NoSystemResponse
	//       403: NoSystemResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/system/actions/:cmd", pRun)
}
