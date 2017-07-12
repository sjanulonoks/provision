package frontend

import (
	"net/http"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
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

// ProfileParamsResponse return on a successful GET of all Profile's Params
// swagger:response
type ProfileParamsResponse struct {
	// in: body
	Body map[string]interface{}
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
// swagger:parameters putProfiles getProfile putProfile patchProfile deleteProfile getProfileParams postProfileParams
type ProfilePathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// ProfileParamsBodyParameter used to set Profile Params
// swagger:parameters postProfileParams
type ProfileParamsBodyParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// ProfileListPathParameter used to limit lists of Profile by path options
// swagger:parameters listProfiles
type ProfileListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Name string
}

func (f *Frontend) InitProfileApi() {
	// swagger:route GET /profiles Profiles listProfiles
	//
	// Lists Profiles filtered by some parameters.
	//
	// This will show all Profiles by default.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Name = string
	//
	// Functions:
	//    Eq(value) = Return items that are equal to value
	//    Lt(value) = Return items that are less than value
	//    Lte(value) = Return items that less than or equal to value
	//    Gt(value) = Return items that are greater than value
	//    Gte(value) = Return items that greater than or equal to value
	//    Between(lower,upper) = Return items that are inclusively between lower and upper
	//    Except(lower,upper) = Return items that are not inclusively between lower and upper
	//
	// Example:
	//    Name=fred - returns items named fred
	//    Name=Lt(fred) - returns items that alphabetically less than fred.
	//
	// Responses:
	//    200: ProfilesResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
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
	//       406: ErrorResponse
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

	// swagger:route GET /profiles/{name}/params Profiles getProfileParams
	//
	// List profile params Profile
	//
	// List Profile parms for a Profile specified by {name}
	//
	//     Responses:
	//       200: ProfileParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/profiles/:name/params",
		func(c *gin.Context) {
			name := c.Param(`name`)
			var res store.KeySaver
			tp := f.dt.NewProfile()
			func() {
				d, unlocker := f.dt.LockEnts(store.KeySaver(tp).(Lockable).Locks("get")...)
				defer unlocker()
				res = d("profiles").Find(name)
			}()
			if res == nil {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: "profiles",
					Key:   name,
				}
				err.Errorf("%s GET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			if !assureAuth(c, f.Logger, res.Prefix(), "get", res.Key()) {
				return
			}
			p := backend.AsProfile(res).GetParams()
			c.JSON(http.StatusOK, p)
		})

	// swagger:route POST /profiles/{name}/params Profiles postProfileParams
	//
	// Set/Replace all the Parameters for a profile specified by {name}
	//
	//     Responses:
	//       200: ProfileParamsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	f.ApiGroup.POST("/profiles/:name/params",
		func(c *gin.Context) {
			var val map[string]interface{}
			if !assureDecode(c, &val) {
				return
			}
			name := c.Param(`name`)
			var res store.KeySaver
			tp := f.dt.NewProfile()
			func() {
				d, unlocker := f.dt.LockEnts(store.KeySaver(tp).(Lockable).Locks("get")...)
				defer unlocker()
				res = d("profiles").Find(name)
			}()
			if res == nil {
				err := &backend.Error{
					Code:  http.StatusNotFound,
					Type:  "API_ERROR",
					Model: "profiles",
					Key:   name,
				}
				err.Errorf("%s SET Params: %s: Not Found", err.Model, err.Key)
				c.JSON(err.Code, err)
				return
			}
			if !assureAuth(c, f.Logger, res.Prefix(), "get", res.Key()) {
				return
			}
			m := backend.AsProfile(res)
			var err error
			func() {
				d, unlocker := f.dt.LockEnts(res.(Lockable).Locks("update")...)
				defer unlocker()
				err = m.SetParams(d, val)
			}()
			if err != nil {
				be, _ := err.(*backend.Error)
				c.JSON(be.Code, be)
			} else {
				c.JSON(http.StatusOK, val)
			}
		})

}
