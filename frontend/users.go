package frontend

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/store"
	"github.com/digitalrebar/provision/backend"
	"github.com/gin-gonic/gin"
)

// UserResponse returned on a successful GET, PUT, PATCH, or POST of a single user
// swagger:response
type UserResponse struct {
	// in: body
	Body *backend.User
}

// UsersResponse returned on a successful GET of all the users
// swagger:response
type UsersResponse struct {
	//in: body
	Body []*backend.User
}

// UserTokenResponse returned on a successful GET of user token
// swagger:response UserTokenResponse
type UserTokenResponse struct {
	//in: body
	Body UserToken
}

// swagger:model
type UserToken struct {
	Token string
	Info  Info
}

// swagger:model
type UserPassword struct {
	Password string
}

// UserBodyParameter used to inject a User
// swagger:parameters createUser putUser
type UserBodyParameter struct {
	// in: body
	// required: true
	Body *backend.User
}

// UserPatchBodyParameter used to patch a User
// swagger:parameters patchUser
type UserPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// UserPutPassword used to set the User's password
// swagger:parameters putUserPassword
type UserPutPasswordParameter struct {
	// in: body
	// required: true
	Body UserPassword
}

// UserPathParameter used to name a User in the path
// swagger:parameters getUser putUser patchUser deleteUser getUserToken putUserPassword
type UserPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// swagger:parameters getUserToken
type UserTokenQueryTTLParameter struct {
	// in: query
	TTL int `json:"ttl"`
}

// swagger:parameters getUserToken
type UserTokenQueryScopeParameter struct {
	// in: query
	Scope string `json:"scope"`
}

// swagger:parameters getUserToken
type UserTokenQueryActionParameter struct {
	// in: query
	Action string `json:"action"`
}

// swagger:parameters getUserToken
type UserTokenQuerySpecificParameter struct {
	// in: query
	Specific string `json:"specific"`
}

// UserListPathParameter used to limit lists of User by path options
// swagger:parameters listUsers
type UserListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Name string
}

func (f *Frontend) InitUserApi(drpid string) {
	// swagger:route GET /users Users listUsers
	//
	// Lists Users filtered by some parameters.
	//
	// This will show all Users by default.
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
	//    200: UsersResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/users",
		func(c *gin.Context) {
			f.List(c, f.dt.NewUser())
		})

	// swagger:route POST /users Users createUser
	//
	// Create a User
	//
	// Create a User from the provided object
	//
	//     Responses:
	//       201: UserResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/users",
		func(c *gin.Context) {
			b := f.dt.NewUser()
			f.Create(c, b, nil)
		})

	// swagger:route GET /users/{name} Users getUser
	//
	// Get a User
	//
	// Get the User specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: UserResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/users/:name",
		func(c *gin.Context) {
			f.Fetch(c, f.dt.NewUser(), c.Param(`name`))
		})

	// swagger:route GET /users/{name}/token Users getUserToken
	//
	// Get a User Token
	//
	// Get a token for the User specified by {name} or return error
	//
	//     Responses:
	//       200: UserTokenResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/users/:name/token",
		func(c *gin.Context) {
			var user store.KeySaver
			func() {
				d, unlocker := f.dt.LockEnts(f.dt.NewUser().Locks("get")...)
				defer unlocker()
				user = d("users").Find(c.Param("name"))
			}()
			if user == nil {
				s := fmt.Sprintf("%s GET: %s: Not Found", "User", c.Param(`name`))
				c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusNotFound, s))
				return
			}
			if !assureAuth(c, f.Logger, user.Prefix(), "token", user.Key()) {
				return
			}
			sttl, _ := c.GetQuery("ttl")
			ttl := 3600
			if sttl != "" {
				ttl64, _ := strconv.ParseInt(sttl, 10, 64)
				ttl = int(ttl64)
			}
			scope, _ := c.GetQuery("scope")
			if scope == "" {
				scope = "*"
			}
			action, _ := c.GetQuery("action")
			if action == "" {
				action = "*"
			}
			specific, _ := c.GetQuery("specific")
			if specific == "" {
				specific = "*"
			}

			if t, err := f.dt.NewToken(c.Param(`name`), ttl, scope, action, specific); err != nil {
				ne, ok := err.(*backend.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				// Error is only if stats are not filled in.  User
				// Token should work regardless of that.
				info, _ := f.GetInfo(drpid)
				c.JSON(http.StatusOK, UserToken{Token: t, Info: *info})
			}
		})

	// swagger:route PATCH /users/{name} Users patchUser
	//
	// Patch a User
	//
	// Update a User specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: UserResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/users/:name",
		func(c *gin.Context) {
			f.Patch(c, f.dt.NewUser(), c.Param(`name`), nil)
		})

	// swagger:route PUT /users/{name} Users putUser
	//
	// Put a User
	//
	// Update a User specified by {name} using a JSON User
	//
	//     Responses:
	//       200: UserResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/users/:name",
		func(c *gin.Context) {
			f.Update(c, f.dt.NewUser(), c.Param(`name`), nil)
		})

	// swagger:route PUT /users/{name}/password Users putUserPassword
	//
	// Set the password for a user.
	//
	// Update a User specified by {name} using a JSON User
	//
	//     Responses:
	//       200: UserResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/users/:name/password",
		func(c *gin.Context) {
			d, unlocker := f.dt.LockEnts(f.dt.NewUser().Locks("update")...)
			defer unlocker()
			obj := d("users").Find(c.Param("name"))
			if obj == nil {
				s := fmt.Sprintf("%s GET: %s: Not Found", "User", c.Param(`name`))
				c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusNotFound, s))
				return
			}
			user, _ := obj.(*backend.User)
			if !assureAuth(c, f.Logger, user.Prefix(), "password", user.Key()) {
				return
			}
			var userPassword UserPassword
			if !assureDecode(c, &userPassword) {
				return
			}
			if err := user.ChangePassword(d, userPassword.Password); err != nil {
				be, ok := err.(*backend.Error)
				if ok {
					c.JSON(be.Code, be)
				} else {
					c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
				}
			} else {
				c.JSON(http.StatusOK, user.Sanitize())
			}
		})

	// swagger:route DELETE /users/{name} Users deleteUser
	//
	// Delete a User
	//
	// Delete a User specified by {name}
	//
	//     Responses:
	//       200: UserResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/users/:name",
		func(c *gin.Context) {
			b := f.dt.NewUser()
			b.Name = c.Param(`name`)
			f.Remove(c, b, nil)
		})
}
