package frontend

import (
	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
)

// ProfileResponse returned on a successful GET, PUT, PATCH, or POST of a single profile
// swagger:response
type ProfileResponse struct {
	// in: body
	Body *backend.Profile
}

// ProfilesResponse returned on a successful GET of all the profiles
// swagger:response
type ProfilesResponse struct {
	//in: body
	Body []*backend.Profile
}

// ProfileBodyParameter used to inject a Profile
// swagger:parameters createProfile putProfile
type ProfileBodyParameter struct {
	// in: body
	// required: true
	Body *backend.Profile
}

// ProfilePatchBodyParameter used to patch a Profile
// swagger:parameters patchProfile
type ProfilePatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// ProfilePathParameter used to name a Profile in the path
// swagger:parameters putProfiles getProfile putProfile patchProfile deleteProfile
type ProfilePathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

func (f *Frontend) InitProfileApi() {
	// swagger:route GET /profiles Profiles listProfiles
	//
	// Lists Profiles filtered by some parameters.
	//
	// This will show all Profiles by default.
	//
	//     Responses:
	//       200: ProfilesResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	f.ApiGroup.GET("/profiles",
		func(c *gin.Context) {
			f.List(c, f.dt.NewProfile())
		})

	// swagger:route POST /profiles Profiles createProfile
	//
	// Create a Profile
	//
	// Create a Profile from the provided object
	//
	//     Responses:
	//       201: ProfileResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/profiles",
		func(c *gin.Context) {
			b := f.dt.NewProfile()
			f.Create(c, b)
		})
	// swagger:route GET /profiles/{name} Profiles getProfile
	//
	// Get a Profile
	//
	// Get the Profile specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: ProfileResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/profiles/:name",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewProfile(), c.Param(`name`))
		})

	// swagger:route PATCH /profiles/{name} Profiles patchProfile
	//
	// Patch a Profile
	//
	// Update a Profile specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: ProfileResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/profiles/:name",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewProfile(), c.Param(`name`))
		})

	// swagger:route PUT /profiles/{name} Profiles putProfile
	//
	// Put a Profile
	//
	// Update a Profile specified by {name} using a JSON Profile
	//
	//     Responses:
	//       200: ProfileResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/profiles/:name",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewProfile(), c.Param(`name`))
		})

	// swagger:route DELETE /profiles/{name} Profiles deleteProfile
	//
	// Delete a Profile
	//
	// Delete a Profile specified by {name}
	//
	//     Responses:
	//       200: ProfileResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/profiles/:name",
		func(c *gin.Context) {
			b := f.dt.NewProfile()
			b.Name = c.Param(`name`)
			f.Remove(c, b)

		})
}
