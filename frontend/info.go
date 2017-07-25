package frontend

import (
	"net/http"
	"runtime"

	"github.com/digitalrebar/provision"
	"github.com/gin-gonic/gin"
)

type Info struct {
	Arch    string `json:"arch"`
	Os      string `json:"os"`
	Version string `json:"version"`
	Id      string `json:"id"`
}

// InfosResponse returned on a successful GET of an info
// swagger:response
type InfoResponse struct {
	// in: body
	Body *Info
}

func (f *Frontend) InitInfoApi(drpid string) {
	// swagger:route GET /info Info getInfo
	//
	// Return current system info.
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       200: InfoResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/info",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "info", "get", "") {
				return
			}

			info := &Info{
				Arch:    runtime.GOARCH,
				Os:      runtime.GOOS,
				Version: provision.RS_VERSION,
				Id:      drpid,
			}

			c.JSON(http.StatusOK, info)
		})
}
