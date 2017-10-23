package frontend

import (
	"fmt"
	"net/http"

	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// InterfacesResponse returned on a successful GET of an interfaces
// swagger:response
type InterfaceResponse struct {
	// in: body
	Body *models.Interface
}

// InterfacesResponse returned on a successful GET of all interfaces
// swagger:response
type InterfacesResponse struct {
	// in: body
	Body []*models.Interface
}

// swagger:parameters getInterface
type InterfaceParameter struct {
	// in: path
	Name string `json:"name"`
}

func (f *Frontend) InitInterfaceApi() {
	// swagger:route GET /interfaces Interfaces listInterfaces
	//
	// Lists possible interfaces on the system to serve DHCP
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       200: InterfacesResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/interfaces",
		func(c *gin.Context) {
			if !f.assureAuth(c, "interfaces", "list", "") {
				return
			}
			intfs, err := f.dt.GetInterfaces()
			if err != nil {
				c.JSON(http.StatusInternalServerError,
					models.NewError(c.Request.Method, http.StatusInternalServerError,
						fmt.Sprintf("interfaces list: %v", err)))
				return
			}
			c.JSON(http.StatusOK, intfs)
		})

	// swagger:route GET /interfaces/{name} Interfaces getInterface
	//
	// Get a specific interface with {name}
	//
	// Get a specific interface specified by {name}.
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       200: InterfaceResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/interfaces/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			if !f.assureAuth(c, "interfaces", "get", name) {
				return
			}
			err := &models.Error{
				Model: "interfaces",
				Key:   name,
				Type:  c.Request.Method,
			}
			intfs, getErr := f.dt.GetInterfaces()
			if getErr != nil {
				err.Code = http.StatusInternalServerError
				err.Errorf("Cannot get interfaces")
				c.JSON(err.Code, err)
				return
			}

			for _, ii := range intfs {
				if ii.Name == c.Param(`name`) {
					c.JSON(http.StatusOK, ii)
					return
				}
			}
			err.Code = http.StatusNotFound
			err.Errorf("No interface")
			c.JSON(err.Code, err)
		})
}
