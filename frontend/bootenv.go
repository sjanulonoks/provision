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
// swagger:parameters createBootEnvs putBootEnv
type BootEnvBodyParameter struct {
	// in: body
	// required: true
	Body *backend.BootEnv
}

// operation represents a valid JSON Patch operation as defined by RFC 6902
type JSONPatchOperation struct {
	// All Operations must have an Op.
	//
	// required: true
	// enum: add,remove,replace,move,copy,test
	Op string `json:"op"`

	// Path is a JSON Pointer as defined in RFC 6901
	// required: true
	Path string `json:"path"`

	// From is a JSON pointer indicating where a value should be
	// copied/moved from.  From is only used by copy and move operations.
	From string `json:"from"`

	// Value is the Value to be used for add, replace, and test operations.
	Value interface{} `json:"value"`
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
	Name string
}

func (f *Frontend) InitBootEnvApi() {
	// swagger:route GET /bootenvs BootEnvs listBootEnvs
	//
	// Lists BootEnvs filtered by some parameters.
	//
	// This will show all BootEnvs by default.
	//
	//     Responses:
	//       default: ErrorResponse
	//       200: BootEnvsResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/bootenvs",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, backend.AsBootEnvs(f.DataTracker.FetchAll(f.DataTracker.NewBootEnv())))
		})

	// swagger:route POST /bootenvs BootEnvs createBootEnv
	//
	// Create a BootEnv
	//
	// Create a BootEnv from the provided object
	//
	//     Responses:
	//       default: ErrorResponse
	//       201: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	f.ApiGroup.POST("/bootenvs",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, err)
			}
			nb, err := f.DataTracker.Create(b)
			if err != nil {
				c.JSON(http.StatusBadRequest, err)
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
	//       default: ErrorResponse
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	f.ApiGroup.GET("/bootenvs/:name",
		func(c *gin.Context) {
			res, ok := f.DataTracker.FetchOne(f.DataTracker.NewBootEnv(), c.Param(`name`))
			if ok {
				c.JSON(http.StatusOK, backend.AsBootEnv(res))
			} else {
				c.JSON(http.StatusNotFound,
					backend.NewError("API ERROR", http.StatusNotFound,
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
	//       default: ErrorResponse
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
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
	//       default: ErrorResponse
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	f.ApiGroup.PUT("/bootenvs/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			if err := c.Bind(b); err != nil {
				c.JSON(http.StatusBadRequest, err)
			}
			if b.Name != c.Param(`name`) {
				c.JSON(http.StatusBadRequest,
					backend.NewError("API ERROR", http.StatusBadRequest,
						fmt.Sprintf("bootenv put: error can not change name: %v %v", c.Param(`name`), b.Name)))
			}
			nb, err := f.DataTracker.Update(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err)
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
	//       default: ErrorResponse
	//       200: BootEnvResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	f.ApiGroup.DELETE("/bootenvs/:name",
		func(c *gin.Context) {
			b := f.DataTracker.NewBootEnv()
			b.Name = c.Param(`name`)
			nb, err := f.DataTracker.Remove(b)
			if err != nil {
				c.JSON(http.StatusNotFound, err)
			} else {
				c.JSON(http.StatusOK, nb)
			}
		})
}
