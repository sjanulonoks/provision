package frontend

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
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
	//        401: ErrorResponse
	f.ApiGroup.GET("/prefs",
		func(c *gin.Context) {
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
	//       401: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.POST("/prefs",
		func(c *gin.Context) {
			prefs := map[string]string{}
			if !assureDecode(c, &prefs) {
				return
			}
			err := &backend.Error{Type: "API_ERROR", Key: "Preference", Code: http.StatusBadRequest}
			// Filter unknown preferences here
			for k := range prefs {
				switch k {
				case "defaultBootEnv", "unknownBootEnv":
					continue
				default:
					err.Errorf("Unknown Preference %s", k)
				}
			}
			err.Merge(f.dt.SetPrefs(prefs))
			if err.ContainsError() {
				c.Error(err)
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusCreated, prefs)
			}
		})
}
