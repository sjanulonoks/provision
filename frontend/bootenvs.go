package frontend

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

// BootEnvResponse returned on a successful GET, PUT, PATCH, or POST of a single bootenv
// swagger:response
type BootEnvResponse struct {
	// in: body
	Body *backend.BootEnv
}

// BootEnvsResponse returned on a successful GET of all the bootenvs
// swagger:response
type BootEnvsResponse struct {
	//in: body
	Body []*backend.BootEnv
}

// BootEnvBodyParameter used to inject a BootEnv
// swagger:parameters createBootEnv putBootEnv
type BootEnvBodyParameter struct {
	// in: body
	// required: true
	Body *backend.BootEnv
}

// BootEnvPatchBodyParameter used to patch a BootEnv
// swagger:parameters patchBootEnv
type BootEnvPatchBodyParameter struct {
	// in: body
	// required: true
	Body []JSONPatchOperation
}

// BootEnvPathParameter used to name a BootEnv in the path
// swagger:parameters putBootEnvs getBootEnv putBootEnv patchBootEnv deleteBootEnv
type BootEnvPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func (f *Frontend) InitBootEnvApi() {
	// swagger:route GET /bootenvs BootEnvs listBootEnvs
	//
	// Lists BootEnvs filtered by some parameters.
	//
	// This will show all BootEnvs by default.
	//
	//     Responses:
	//       200: BootEnvsResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/bootenvs",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsBootEnvs(f.dt.FetchAll(f.dt.NewBootEnv())))
		})

	// swagger:route POST /bootenvs BootEnvs createBootEnv
	//
	// Create a BootEnv
	//
	// Create a BootEnv from the provided object
	//
	//     Responses:
	//       201: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/bootenvs",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			nb, err := f.dt.Create(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusCreated, nb)
			}
		})

	// swagger:route GET /bootenvs/{name} BootEnvs getBootEnv
	//
	// Get a BootEnv
	//
	// Get the BootEnv specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: BootEnvResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/bootenvs/:name",
		func(c *gin.Context) {
			res, ok := f.dt.FetchOne(f.dt.NewBootEnv(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, backend.AsBootEnv(res))
			} else {
				c.JSON(http.StatusNotFound,
					backend.NewError("API_ERROR", http.StatusNotFound,
						fmt.Sprintf("bootenv get: error not found: %v", c.Param(`name`))))
			}
		})

	// swagger:route PATCH /bootenvs/{name} BootEnvs patchBootEnv
	//
	// Patch a BootEnv
	//
	// Update a BootEnv specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/bootenvs/:name",
		func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, backend.NewError("API_ERROR", http.StatusNotImplemented, "bootenv patch: NOT IMPLEMENTED"))
		})

	// swagger:route PUT /bootenvs/{name} BootEnvs putBootEnv
	//
	// Put a BootEnv
	//
	// Update a BootEnv specified by {name} using a JSON BootEnv
	//
	//     Responses:
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/bootenvs/:name",
		func(c *gin.Context) {
			if !testContentType(c, "application/json") {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, fmt.Sprintf("Invalid content type: %s", c.ContentType())))
				return
			}
			b := f.dt.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				return
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API_ERROR", http.StatusBadRequest,
						fmt.Sprintf("bootenv put: error can not change name: %v %v", c.Param(`name`), b.Name)))
				return
			}
			nb, err := f.dt.Update(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})

	// swagger:route DELETE /bootenvs/{name} BootEnvs deleteBootEnv
	//
	// Delete a BootEnv
	//
	// Delete a BootEnv specified by {name}
	//
	//     Responses:
	//       200: BootEnvResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/bootenvs/:name",
		func(c *gin.Context) {
			b := f.dt.NewBootEnv()
			b.Name = c.Param(`name`)
			nb, err := f.dt.Remove(b)
			if err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
}
