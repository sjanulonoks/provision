package frontend

import (
	"net/http"
	"runtime"

	"github.com/digitalrebar/provision"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// InfosResponse returned on a successful GET of an info
// swagger:response
type InfoResponse struct {
	// in: body
	Body *models.Info
}

func (f *Frontend) GetInfo(drpid string) (*models.Info, *models.Error) {
	i := &models.Info{
		Arch:               runtime.GOARCH,
		Os:                 runtime.GOOS,
		Version:            provision.RS_VERSION,
		Id:                 drpid,
		ApiPort:            f.ApiPort,
		FilePort:           f.ProvPort,
		TftpEnabled:        !f.NoTftp,
		DhcpEnabled:        !f.NoDhcp,
		ProvisionerEnabled: !f.NoProv,
		Stats:              make([]*models.Stat, 0, 0),
		Features: []string{
			"api-v3",
			"sane-exit-codes",
			"common-blob-size",
			"change-stage-map",
			"job-exit-states",
			"package-repository-handling",
		},
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
			i.Stats = append(i.Stats, &models.Stat{"machines.count", idx.Count()})
		}

		if idx, err := index.All(index.Native())(&d("subnets").Index); err != nil {
			res.AddError(err)
		} else {
			i.Stats = append(i.Stats, &models.Stat{"subnets.count", idx.Count()})
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
			if !f.assureAuth(c, "info", "get", "") {
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
