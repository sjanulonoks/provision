package frontend

import (
	"net/http"
	"strconv"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// PrefsResponse returned on a successful GET of all preferences
// swagger:response
type PrefsResponse struct {
	// in: body
	Body map[string]string
}

// PrefBodyParameter is used to create or update a Pref
// swagger:parameters setPrefs
type PrefBodyParameter struct {
	// in: body
	Body map[string]string
}

func (f *Frontend) InitPrefApi() {
	// swagger:route GET /prefs Prefs listPrefs
	//
	// Lists Prefs
	//
	// This will show all Prefs by default
	//
	//      Responses:
	//        200: PrefsResponse
	//        401: NoContentResponse
	//        403: NoContentResponse
	f.ApiGroup.GET("/prefs",
		func(c *gin.Context) {
			if !f.assureAuth(c, "prefs", "list", "") {
				return
			}
			c.JSON(http.StatusOK, f.dt.Prefs())
		})

	// swagger:route POST /prefs Prefs setPrefs
	//
	// Create a Pref
	//
	// Create a Pref from the provided object
	//
	//      Responses:
	//       201: PrefsResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/prefs",
		func(c *gin.Context) {
			prefs := map[string]string{}
			if !assureDecode(c, &prefs) {
				return
			}
			err := &models.Error{Type: "API_ERROR", Key: "Preference", Code: http.StatusBadRequest}
			// Filter unknown preferences here
			for k := range prefs {
				switch k {
				case "baseTokenSecret":
					if !f.assureAuth(c, "prefs", "post", k) {
						return
					}
					if len(prefs[k]) != 32 {
						err.Errorf("Preference %s: Must be 32 bytes long", k)
					}
				case "defaultBootEnv", "unknownBootEnv", "defaultStage", "systemGrantorSecret":
					if !f.assureAuth(c, "prefs", "post", k) {
						return
					}
				case "knownTokenTimeout", "unknownTokenTimeout", "debugRenderer", "debugDhcp", "debugBootEnv", "debugFrontend", "debugPlugins":
					if !f.assureAuth(c, "prefs", "post", k) {
						return
					}
					if _, e := strconv.Atoi(prefs[k]); e != nil {
						err.Errorf("Preference %s: %v", k, e)
					}
				default:
					err.Errorf("Unknown Preference %s", k)
				}
			}
			if !err.ContainsError() {
				func() {
					d, unlocker := f.dt.LockEnts(models.Model(&backend.Pref{}).(Lockable).Locks("update")...)
					defer unlocker()
					err.AddError(f.dt.SetPrefs(d, prefs))
				}()
			}
			if err.ContainsError() {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusCreated, f.dt.Prefs())
			}
		})
}
