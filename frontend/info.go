package frontend

import (
	"net/http"
	"runtime"

	"github.com/digitalrebar/provision"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

type Stat struct {
	// required: true
	Name string `json:"name"`
	// required: true
	Count int `json:"count"`
}

// swagger:model
type Info struct {
	// required: true
	Arch string `json:"arch"`
	// required: true
	Os string `json:"os"`
	// required: true
	Version string `json:"version"`
	// required: true
	Id string `json:"id"`
	// required: true
	ApiPort int `json:"api_port"`
	// required: true
	FilePort int `json:"file_port"`
	// required: true
	TftpEnabled bool `json:"tftp_enabled"`
	// required: true
	DhcpEnabled bool `json:"dhcp_enabled"`
	// required: true
	ProvisionerEnabled bool `json:"prov_enabled"`
	// required: true
	Stats []*Stat `json:"stats"`
}

// InfosResponse returned on a successful GET of an info
// swagger:response
type InfoResponse struct {
	// in: body
	Body *Info
}

func (f *Frontend) GetInfo(drpid string) (*Info, *models.Error) {
	i := &Info{
		Arch:               runtime.GOARCH,
		Os:                 runtime.GOOS,
		Version:            provision.RS_VERSION,
		Id:                 drpid,
		ApiPort:            f.ApiPort,
		FilePort:           f.ProvPort,
		TftpEnabled:        !f.NoTftp,
		DhcpEnabled:        !f.NoDhcp,
		ProvisionerEnabled: !f.NoProv,
		Stats:              make([]*Stat, 0, 0),
	}

	res := &models.Error{
		Code:  http.StatusInternalServerError,
		Type:  "API_ERROR",
		Model: "info",
	}

	func() {
		d, unlocker := f.dt.LockEnts("machines", "subnets")
		defer unlocker()

		if idx, err := index.All(index.Native())(&d("machines").Index); err != nil {
			res.AddError(err)
		} else {
			i.Stats = append(i.Stats, &Stat{"machines.count", idx.Count()})
		}

		if idx, err := index.All(index.Native())(&d("subnets").Index); err != nil {
			res.AddError(err)
		} else {
			i.Stats = append(i.Stats, &Stat{"subnets.count", idx.Count()})
		}
	}()

	if res.HasError() == nil {
		res = nil
	}

	return i, res
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
			info, err := f.GetInfo(drpid)
			if err != nil {
				c.JSON(err.Code, err)
				return
			}
			c.JSON(http.StatusOK, info)
		})
}
