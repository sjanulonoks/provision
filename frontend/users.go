package frontend

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// UserResponse returned on a successful GET, PUT, PATCH, or POST of a single user
// swagger:response
type UserResponse struct {
	// in: body
	Body *models.User
}

// UsersResponse returned on a successful GET of all the users
// swagger:response
type UsersResponse struct {
	//in: body
	Body []*models.User
}

// UserTokenResponse returned on a successful GET of user token
// swagger:response UserTokenResponse
type UserTokenResponse struct {
	//in: body
	Body models.UserToken
}

// UserBodyParameter used to inject a User
// swagger:parameters createUser putUser
type UserBodyParameter struct {
	// in: body
	// required: true
	Body *models.User
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
	Body models.UserPassword
}

// UserPathParameter used to name a User in the path
// swagger:parameters getUser putUser patchUser deleteUser getUserToken putUserPassword headUser
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
// swagger:parameters listUsers listStatsUsers
type UserListPathParameter struct {
	// in: query
	Offest int `json:"offset"`
	// in: query
	Limit int `json:"limit"`
	// in: query
	Available string
	// in: query
	Valid string
	// in: query
	ReadOnly string
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
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
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
			f.List(c, &backend.User{})
		})

	// swagger:route HEAD /users Users listStatsUsers
	//
	// Stats of the List Users filtered by some parameters.
	//
	// This will return headers with the stats of the list.
	//
	// You may specify:
	//    Offset = integer, 0-based inclusive starting point in filter data.
	//    Limit = integer, number of items to return
	//
	// Functional Indexs:
	//    Name = string
	//    Available = boolean
	//    Valid = boolean
	//    ReadOnly = boolean
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
	//    200: NoContentResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.HEAD("/users",
		func(c *gin.Context) {
			f.ListStats(c, &backend.User{})
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/users",
		func(c *gin.Context) {
			b := &backend.User{}
			f.Create(c, b)
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
			f.Fetch(c, &backend.User{}, c.Param(`name`))
		})

	// swagger:route HEAD /users/{name} Users headUser
	//
	// See if a User exists
	//
	// Return 200 if the User specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/users/:name",
		func(c *gin.Context) {
			f.Exists(c, &backend.User{}, c.Param(`name`))
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
			ref := &backend.User{}
			var userName, grantorName, userSecret, grantorSecret string
			var err *models.Error
			rt := f.rt(c, ref.Locks("get")...)
			rt.Do(func(d backend.Stores) {
				err = &models.Error{
					Type:  c.Request.Method,
					Code:  http.StatusNotFound,
					Model: "users",
					Key:   c.Param("name"),
				}
				u := rt.Find("users", c.Param("name"))
				g := rt.Find("users", f.getAuthUser(c))
				if u == nil || g == nil {
					err.Errorf("Not Found")
					return
				}
				uobj := backend.AsUser(u)
				gobj := backend.AsUser(g)
				userName, userSecret = uobj.Name, uobj.Secret
				grantorName, grantorSecret = gobj.Name, gobj.Secret
				err = nil
			})
			if err != nil {
				c.JSON(err.Code, err)
				return
			}
			if !f.assureAuth(c, "users", "token", userName) {
				return
			}
			sttl, _ := c.GetQuery("ttl")
			ttl := time.Hour
			if sttl != "" {
				ttl64, err := strconv.ParseInt(sttl, 10, 64)
				if err != nil {
					res := &models.Error{
						Type:  c.Request.Method,
						Model: "users",
						Key:   c.Param(`name`),
						Code:  http.StatusBadRequest,
					}
					res.AddError(err)
					c.JSON(res.Code, res)
					return
				}
				ttl = time.Second * time.Duration(ttl64)
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

			claims := backend.NewClaim(c.Param(`name`), grantorName, ttl).
				Add(scope, action, specific).
				AddSecrets(grantorSecret, userSecret, "")

			if t, err := f.dt.SealClaims(claims); err != nil {
				ne, ok := err.(*models.Error)
				if ok {
					c.JSON(ne.Code, ne)
				} else {
					c.JSON(http.StatusBadRequest, models.NewError(c.Request.Method, http.StatusBadRequest, err.Error()))
				}
			} else {
				// Error is only if stats are not filled in.  User
				// Token should work regardless of that.
				info, _ := f.GetInfo(c, drpid)
				if info != nil {
					if a, _, e := net.SplitHostPort(c.Request.RemoteAddr); e == nil {
						info.Address = backend.LocalFor(net.ParseIP(a))
					}
				}
				c.JSON(http.StatusOK, models.UserToken{Token: t, Info: *info})
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/users/:name",
		func(c *gin.Context) {
			f.Patch(c, &backend.User{}, c.Param(`name`))
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/users/:name",
		func(c *gin.Context) {
			f.Update(c, &backend.User{}, c.Param(`name`))
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
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/users/:name/password",
		func(c *gin.Context) {
			if !f.assureAuth(c, "users", "password", c.Param("name")) {
				return
			}
			var userPassword models.UserPassword
			if !assureDecode(c, &userPassword) {
				return
			}
			var user *models.User
			var err *models.Error
			ref := &backend.User{}
			rt := f.rt(c, ref.Locks("update")...)
			rt.Do(func(d backend.Stores) {
				res := &models.Error{
					Type:  c.Request.Method,
					Model: "users",
					Key:   c.Param(`name`),
					Code:  http.StatusNotFound,
				}
				obj := rt.Find("users", c.Param("name"))
				if obj == nil {
					res.Errorf("Not Found")
					err = res
					return
				}
				rUser := backend.AsUser(obj)
				if uErr := rUser.ChangePassword(rt, userPassword.Password); uErr != nil {
					res.Code = http.StatusBadRequest
					res.AddError(uErr)
					err = res
					return
				}
				user = models.Clone(rUser.User).(*models.User)
			})
			if err != nil {
				c.JSON(err.Code, err)
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
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/users/:name",
		func(c *gin.Context) {
			f.Remove(c, &backend.User{}, c.Param(`name`))
		})
}
