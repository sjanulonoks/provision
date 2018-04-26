package frontend

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

func (f *Frontend) makeParamEndpoints(obj models.Paramer, idKey string) (
	getAll, getOne, patchThem, setThem, setOne, deleteOne func(c *gin.Context)) {
	trimmer := func(s string) string {
		return strings.TrimLeft(s, `/`)
	}
	aggregator := func(c *gin.Context) bool {
		return c.Query("aggregate") == "true"
	}
	item404 := func(c *gin.Context, found bool, key, line string) bool {
		if !found {
			err := &models.Error{
				Code:  http.StatusNotFound,
				Type:  c.Request.Method,
				Model: obj.Prefix(),
				Key:   key,
			}
			err.Errorf("Not Found")
			c.JSON(err.Code, err)
		}
		return !found
	}
	idrtkeyok := func(c *gin.Context, op string) (string, *backend.RequestTracker, string) {
		id := c.Param(idKey)
		return id,
			f.rt(c, obj.(Lockable).Locks(op)...),
			trimmer(c.Param("key"))
	}
	updater := func(c *gin.Context,
		rt *backend.RequestTracker,
		orig, changed models.Paramer) (authed bool, err *models.Error) {
		err = &models.Error{Code: 422, Type: backend.ValidationError, Model: orig.Prefix(), Key: orig.Key()}
		patch, patchErr := models.GenPatch(orig, changed, false)
		if patchErr != nil {
			err.AddError(patchErr)
			return true, err
		}
		if !f.assureAuthUpdate(c, changed.Prefix(), "update", changed.Key(), patch) {
			return false, err
		}
		_, updateErr := rt.Update(changed)
		err.AddError(updateErr)
		return true, err
	}
	return /* getAll */ func(c *gin.Context) {
			id, rt, _ := idrtkeyok(c, "get")
			if !f.assureSimpleAuth(c, obj.Prefix(), "get", id) {
				return
			}
			var params map[string]interface{}
			var found bool
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob != nil {
					params, found = rt.GetParams(ob.(models.Paramer), aggregator(c)), true
				}
			})
			if !item404(c, found, id, "Params") {
				c.JSON(http.StatusOK, params)
			}
		},
		/* getOne */ func(c *gin.Context) {
			id, rt, key := idrtkeyok(c, "get")
			if !f.assureSimpleAuth(c, obj.Prefix(), "get", id) {
				return
			}
			var found bool
			var val interface{}
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob != nil {
					found = true
					val, _ = rt.GetParam(ob.(models.Paramer), key, aggregator(c))
				}
			})
			if !item404(c, found, id, "Param") {
				c.JSON(http.StatusOK, val)
			}
		},
		/* patchThem */ func(c *gin.Context) {
			var patch jsonpatch2.Patch
			if !assureDecode(c, &patch) {
				return
			}
			id, rt, _ := idrtkeyok(c, "update")
			rt.Tracef("Patching %s:%s with %#v", obj.Prefix(), id, patch)
			var res map[string]interface{}
			var authed, found bool
			var patchErr *models.Error
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob == nil {
					authed = f.assureSimpleAuth(c, obj.Prefix(), "update", id)
					return
				}
				authed = true
				orig := models.Clone(ob).(models.Paramer)
				changed := models.Clone(ob).(models.Paramer)
				params := orig.GetParams()
				rt.Tracef("Object %s:%s exists, has params %#v", orig.Prefix(), id, params)
				if params == nil {
					params = map[string]interface{}{}
				}

				found = true
				patchErr = &models.Error{
					Code:  http.StatusConflict,
					Type:  c.Request.Method,
					Model: orig.Prefix(),
					Key:   id,
				}
				buf, err := json.Marshal(params)
				if err != nil {
					patchErr.AddError(err)
					return
				}
				patched, err, loc := patch.Apply(buf)
				if err != nil {
					patchErr.Errorf("Patch failed to apply at line %d", loc)
					patchErr.AddError(err)
					return
				}
				if err := json.Unmarshal(patched, &res); err != nil {
					patchErr.AddError(err)
				}
				if !patchErr.ContainsError() {
					changed.SetParams(res)
					authed, err = updater(c, rt, orig, changed)
					if err != nil {
						patchErr.AddError(err)
					}
				}
			})
			if !authed {
				return
			}
			if item404(c, found, id, "Params") {
				return
			}
			if patchErr.ContainsError() {
				c.JSON(patchErr.Code, patchErr)
			} else {
				c.JSON(http.StatusOK, res)
			}
		},
		/* setThem */ func(c *gin.Context) {
			id, rt, _ := idrtkeyok(c, "update")
			var replacement map[string]interface{}
			if !assureDecode(c, &replacement) {
				return
			}
			var authed, found bool
			var err *models.Error
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob == nil {
					authed = f.assureSimpleAuth(c, obj.Prefix(), "update", id)
					return
				}
				changed := models.Clone(ob).(models.Paramer)
				orig := models.Clone(ob).(models.Paramer)
				found = true
				changed.SetParams(replacement)
				authed, err = updater(c, rt, orig, changed)
			})
			if !authed {
				return
			}
			if item404(c, found, id, "Params") {
				return
			}
			if err != nil && err.ContainsError() {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, replacement)
			}
		},
		/* setOne */ func(c *gin.Context) {
			id, rt, key := idrtkeyok(c, "update")
			var replacement interface{}
			if !assureDecode(c, &replacement) {
				return
			}
			var authed, found bool
			var err *models.Error
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob == nil {
					authed = f.assureSimpleAuth(c, obj.Prefix(), "update", id)
					return
				}
				found = true
				changed := models.Clone(ob).(models.Paramer)
				orig := models.Clone(ob).(models.Paramer)
				params := orig.GetParams()
				params[key] = replacement
				changed.SetParams(params)
				authed, err = updater(c, rt, orig, changed)
			})
			if !authed {
				return
			}
			if item404(c, found, id, "Params") {
				return
			}
			if err != nil && err.ContainsError() {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, replacement)
			}
		},
		/* deleteOne */ func(c *gin.Context) {
			id, rt, key := idrtkeyok(c, "update")
			var authed, found bool
			var val interface{}
			var err *models.Error
			rt.Do(func(d backend.Stores) {
				ob := rt.Find(obj.Prefix(), id)
				if ob == nil {
					authed = f.assureSimpleAuth(c, obj.Prefix(), "update", id)
					return
				}
				found = true
				changed := models.Clone(ob).(models.Paramer)
				orig := models.Clone(ob).(models.Paramer)
				params := orig.GetParams()
				valFound := false
				val, valFound = params[key]
				delete(params, key)
				changed.SetParams(params)
				authed, err = updater(c, rt, orig, changed)
				if !valFound {
					err = &models.Error{
						Code:  http.StatusNotFound,
						Type:  "DELETE",
						Model: "params",
						Key:   key,
					}
					err.Errorf("Not Found")
				}
			})
			if !authed {
				return
			}
			if item404(c, found, id, "Params") {
				return
			}
			if err != nil && err.ContainsError() {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, val)
			}
		}
}

// ParamResponse returned on a successful GET, PUT, PATCH, or POST of a single param
// swagger:response
type ParamResponse struct {
	// in: body
	Body *models.Param
}

// ParamsResponse returned on a successful GET of all the params
// swagger:response
type ParamsResponse struct {
	//in: body
	Body []*models.Param
}

// ParamParamsResponse return on a successful GET of all Param's Params
// swagger:response
type ParamParamsResponse struct {
	// in: body
	Body map[string]interface{}
}

// ParamBodyParameter used to inject a Param
// swagger:parameters createParam putParam
type ParamBodyParameter struct {
	// in: body
	// required: true
	Body *models.Param
}

// ParamPatchBodyParameter used to patch a Param
// swagger:parameters patchParam
type ParamPatchBodyParameter struct {
	// in: body
	// required: true
	Body jsonpatch2.Patch
}

// ParamPathParameter used to name a Param in the path
// swagger:parameters putParams getParam putParam patchParam deleteParam getParamParams postParamParams headParam
type ParamPathParameter struct {
	// in: path
	// required: true
	Name string `json:"name"`
}

// ParamParamsBodyParameter used to set Param Params
// swagger:parameters postParamParams
type ParamParamsBodyParameter struct {
	// in: body
	// required: true
	Body map[string]interface{}
}

// ParamListPathParameter used to limit lists of Param by path options
// swagger:parameters listParams listStatsParams
type ParamListPathParameter struct {
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

func (f *Frontend) InitParamApi() {
	// swagger:route GET /params Params listParams
	//
	// Lists Params filtered by some parameters.
	//
	// This will show all Params by default.
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
	//    200: ParamsResponse
	//    401: NoContentResponse
	//    403: NoContentResponse
	//    406: ErrorResponse
	f.ApiGroup.GET("/params",
		func(c *gin.Context) {
			f.List(c, &backend.Param{})
		})

	// swagger:route HEAD /params Params listStatsParams
	//
	// Stats of the List Params filtered by some parameters.
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
	f.ApiGroup.HEAD("/params",
		func(c *gin.Context) {
			f.ListStats(c, &backend.Param{})
		})

	// swagger:route POST /params Params createParam
	//
	// Create a Param
	//
	// Create a Param from the provided object
	//
	//     Responses:
	//       201: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/params",
		func(c *gin.Context) {
			b := &backend.Param{}
			f.Create(c, b)
		})
	// swagger:route GET /params/{name} Params getParam
	//
	// Get a Param
	//
	// Get the Param specified by {name} or return NotFound.
	//
	//     Responses:
	//       200: ParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/params/*name",
		func(c *gin.Context) {
			name := strings.TrimLeft(c.Param(`name`), `/`)
			f.Fetch(c, &backend.Param{}, name)
		})

	// swagger:route HEAD /params/{name} Params headParam
	//
	// See if a Param exists
	//
	// Return 200 if the Param specifiec by {name} exists, or return NotFound.
	//
	//     Responses:
	//       200: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: NoContentResponse
	f.ApiGroup.HEAD("/params/*name",
		func(c *gin.Context) {
			name := strings.TrimLeft(c.Param(`name`), `/`)
			f.Exists(c, &backend.Param{}, name)
		})

	// swagger:route PATCH /params/{name} Params patchParam
	//
	// Patch a Param
	//
	// Update a Param specified by {name} using a RFC6902 Patch structure
	//
	//     Responses:
	//       200: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       406: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PATCH("/params/*name",
		func(c *gin.Context) {
			name := strings.TrimLeft(c.Param(`name`), `/`)
			f.Patch(c, &backend.Param{}, name)
		})

	// swagger:route PUT /params/{name} Params putParam
	//
	// Put a Param
	//
	// Update a Param specified by {name} using a JSON Param
	//
	//     Responses:
	//       200: ParamResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.PUT("/params/*name",
		func(c *gin.Context) {
			name := strings.TrimLeft(c.Param(`name`), `/`)
			f.Update(c, &backend.Param{}, name)
		})

	// swagger:route DELETE /params/{name} Params deleteParam
	//
	// Delete a Param
	//
	// Delete a Param specified by {name}
	//
	//     Responses:
	//       200: ParamResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/params/*name",
		func(c *gin.Context) {
			name := strings.TrimLeft(c.Param(`name`), `/`)
			f.Remove(c, &backend.Param{}, name)
		})
}
