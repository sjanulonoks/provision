package frontend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/dgrijalva/jwt-go"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
	"github.com/rackn/rocket-skates/embedded"
)

// ErrorResponse is returned whenever an error occurs
// swagger:response
type ErrorResponse struct {
	//in: body
	Body backend.Error
}

// NoContentResponse is returned for deletes and auth errors
// swagger:response
type NoContentResponse struct {
	//description: Nothing
}

// This interface defines the pieces of the backend.DataTracker that the
// frontend needs.
type DTI interface {
	Create(store.KeySaver) (store.KeySaver, error)
	Update(store.KeySaver) (store.KeySaver, error)
	Remove(store.KeySaver) (store.KeySaver, error)
	Save(store.KeySaver) (store.KeySaver, error)
	Patch(store.KeySaver, string, jsonpatch2.Patch) (store.KeySaver, error)
	FetchOne(store.KeySaver, string) (store.KeySaver, bool)
	FetchAll(ref store.KeySaver) []store.KeySaver

	NewBootEnv() *backend.BootEnv
	NewMachine() *backend.Machine
	NewTemplate() *backend.Template
	NewLease() *backend.Lease
	NewReservation() *backend.Reservation
	NewSubnet() *backend.Subnet
	NewUser() *backend.User
	NewParam() *backend.Param

	Pref(string) (string, error)
	Prefs() map[string]string
	SetPrefs(map[string]string) error

	GetInterfaces() ([]*backend.Interface, error)

	GetToken(string) (*backend.DrpCustomClaims, error)
	NewToken(string, int, string, string, string) (string, error)
}

type Frontend struct {
	Logger     *log.Logger
	FileRoot   string
	MgmtApi    *gin.Engine
	ApiGroup   *gin.RouterGroup
	dt         DTI
	authSource AuthSource
}

type AuthSource interface {
	GetUser(username string) *backend.User
}

type DefaultAuthSource struct {
	dt DTI
}

func (d DefaultAuthSource) GetUser(username string) (u *backend.User) {
	userThing, found := d.dt.FetchOne(d.dt.NewUser(), username)
	if !found {
		return
	}
	u = backend.AsUser(userThing)
	return
}

func NewDefaultAuthSource(dt DTI) (das AuthSource) {
	das = DefaultAuthSource{dt: dt}
	return
}

func NewFrontend(dt DTI, logger *log.Logger, fileRoot, devUI string, authSource AuthSource) (me *Frontend) {

	if authSource == nil {
		authSource = NewDefaultAuthSource(dt)
	}

	userAuth := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			// Check for Token Header
			drpToken := c.Request.Header.Get("DRP-AUTH-TOKEN")
			if len(drpToken) != 0 {
				t, err := dt.GetToken(drpToken)
				if err == nil {
					c.Set("DRP-CLAIM", t)
					c.Next()
					return
				} else {
					logger.Printf("No DRP authentication token")
					c.Header("WWW-Authenticate", "rocket-skates")
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
			}

			authHeader := c.Request.Header.Get("Authorization")
			if len(authHeader) == 0 {
				logger.Printf("No authentication header")
				c.Header("WWW-Authenticate", "rocket-skates")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			hdrParts := strings.SplitN(authHeader, " ", 2)
			if len(hdrParts) != 2 || hdrParts[0] != "Basic" {
				logger.Printf("Bad auth header: %s", authHeader)
				c.Header("WWW-Authenticate", "rocket-skates")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			hdr, err := base64.StdEncoding.DecodeString(hdrParts[1])
			if err != nil {
				logger.Printf("Malformed basic auth string: %s", hdrParts[1])
				c.Header("WWW-Authenticate", "rocket-skates")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			userpass := bytes.SplitN(hdr, []byte(`:`), 2)
			if len(userpass) != 2 {
				logger.Printf("Malformed basic auth string: %s", hdrParts[1])
				c.Header("WWW-Authenticate", "rocket-skates")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			user := authSource.GetUser(string(userpass[0]))
			if user == nil {
				logger.Printf("No such user: %s", string(userpass[0]))
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if !user.CheckPassword(string(userpass[1])) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			t := &backend.DrpCustomClaims{
				"all",
				"",
				"",
				jwt.StandardClaims{
					Issuer: "digitalrebar provision",
					Id:     string(userpass[0]),
				},
			}
			c.Set("DRP-CLAIM", t)
			c.Next()
		}
	}

	mgmtApi := gin.Default()
	mgmtApi.Use(userAuth())

	apiGroup := mgmtApi.Group("/api/v3")

	me = &Frontend{Logger: logger, FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, dt: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()
	me.InitLeaseApi()
	me.InitReservationApi()
	me.InitSubnetApi()
	me.InitUserApi()
	me.InitInterfaceApi()
	me.InitPrefApi()
	me.InitParamApi()

	// Swagger.json serve
	buf, err := embedded.Asset("swagger.json")
	if err != nil {
		logger.Fatalf("Failed to load swagger.json asset")
	}
	var f interface{}
	err = json.Unmarshal(buf, &f)
	mgmtApi.GET("/swagger.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, f)
	})

	// Server Swagger UI.
	mgmtApi.StaticFS("/swagger-ui",
		&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "swagger-ui"})

	// Server UI with flag to run from local files instead of assets
	if len(devUI) == 0 {
		mgmtApi.StaticFS("/ui",
			&assetfs.AssetFS{Asset: embedded.Asset, AssetDir: embedded.AssetDir, AssetInfo: embedded.AssetInfo, Prefix: "ui"})
	} else {
		logger.Printf("DEV: Running UI from %s\n", devUI)
		mgmtApi.Static("/ui", devUI)
	}

	// root path, forward to UI
	mgmtApi.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/ui/")
	})

	return
}

func testContentType(c *gin.Context, ct string) bool {
	ct = strings.ToUpper(ct)
	test := strings.ToUpper(c.ContentType())

	return strings.Contains(test, ct)
}

func assureContentType(c *gin.Context, ct string) bool {
	if testContentType(c, ct) {
		return true
	}
	err := &backend.Error{Type: "API_ERROR", Code: http.StatusBadRequest}
	err.Errorf("Invalid content type: %s", c.ContentType())
	c.JSON(err.Code, err)
	return false
}

func assureAuth(c *gin.Context, logger *log.Logger, scope, action, specific string) bool {
	obj, ok := c.Get("DRP-CLAIM")
	if !ok {
		logger.Printf("Request with no claims\n")
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	drpClaim, ok := obj.(*backend.DrpCustomClaims)
	if !ok {
		logger.Printf("Request with bad claims\n")
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	if drpClaim.Scope != "all" && drpClaim.Scope != scope {
		logger.Printf("Request with bad scope: %s, %s\n", scope, drpClaim.Scope)
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	if drpClaim.Action != "" && drpClaim.Action != action {
		logger.Printf("Request with bad action: %s, %s\n", action, drpClaim.Action)
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	if drpClaim.Specific != "" && drpClaim.Specific != specific {
		logger.Printf("Request with bad specific: %s, %s\n", specific, drpClaim.Specific)
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	return true
}

func assureDecode(c *gin.Context, val interface{}) bool {
	if !assureContentType(c, "application/json") {
		return false
	}
	err := &backend.Error{Type: "API_ERROR", Code: http.StatusBadRequest}
	marshalErr := c.Bind(&val)
	if marshalErr == nil {
		return true
	}
	err.Merge(marshalErr)
	c.JSON(err.Code, err)
	return false
}

func (f *Frontend) List(c *gin.Context, ref store.KeySaver) {
	if !assureAuth(c, f.Logger, ref.Prefix(), "list", "") {
		return
	}
	c.JSON(http.StatusOK, f.dt.FetchAll(ref))
}

func (f *Frontend) Fetch(c *gin.Context, ref store.KeySaver, key string) {
	res, ok := f.dt.FetchOne(ref, key)
	if ok {
		if !assureAuth(c, f.Logger, ref.Prefix(), "get", res.Key()) {
			return
		}
		c.JSON(http.StatusOK, res)
	} else {
		err := &backend.Error{
			Code:  http.StatusNotFound,
			Type:  "API_ERROR",
			Model: ref.Prefix(),
			Key:   key,
		}
		err.Errorf("%s GET: %s: Not Found", err.Model, err.Key)
		c.JSON(err.Code, err)
	}
}

func (f *Frontend) Create(c *gin.Context, val store.KeySaver) {
	if !assureDecode(c, val) {
		return
	}
	if !assureAuth(c, f.Logger, val.Prefix(), "create", "") {
		return
	}
	res, err := f.dt.Create(val)
	if err != nil {
		be, ok := err.(*backend.Error)
		if ok {
			c.JSON(be.Code, be)
		} else {
			c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
		}
	} else {
		c.JSON(http.StatusCreated, res)
	}
}

func (f *Frontend) Patch(c *gin.Context, ref store.KeySaver, key string) {
	patch := make(jsonpatch2.Patch, 0)
	if !assureDecode(c, &patch) {
		return
	}
	if !assureAuth(c, f.Logger, ref.Prefix(), "patch", key) {
		return
	}
	res, err := f.dt.Patch(ref, key, patch)
	if err == nil {
		c.JSON(http.StatusOK, res)
		return
	}
	ne, ok := err.(*backend.Error)
	if ok {
		c.JSON(ne.Code, ne)
	} else {
		c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
	}
}

func (f *Frontend) Update(c *gin.Context, ref store.KeySaver, key string) {
	if !assureDecode(c, ref) {
		return
	}
	if !assureAuth(c, f.Logger, ref.Prefix(), "update", key) {
		return
	}
	if ref.Key() != key {
		err := &backend.Error{
			Code:  http.StatusBadRequest,
			Type:  "API_ERROR",
			Model: ref.Prefix(),
			Key:   key,
		}
		err.Errorf("%s PUT: Key change from %s to %s not allowed", err.Model, key, ref.Key())
		c.JSON(err.Code, err)
		return
	}
	newThing, err := f.dt.Update(ref)
	if err == nil {
		c.JSON(http.StatusOK, newThing)
		return
	}
	ne, ok := err.(*backend.Error)
	if ok {
		c.JSON(ne.Code, ne)
	} else {
		c.JSON(http.StatusBadRequest, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
	}
}

func (f *Frontend) Remove(c *gin.Context, ref store.KeySaver) {
	if !assureAuth(c, f.Logger, ref.Prefix(), "delete", ref.Key()) {
		return
	}
	res, err := f.dt.Remove(ref)
	if err != nil {
		ne, ok := err.(*backend.Error)
		if ok {
			c.JSON(ne.Code, ne)
		} else {
			c.JSON(http.StatusNotFound, backend.NewError("API_ERROR", http.StatusBadRequest, err.Error()))
		}
	} else {
		c.JSON(http.StatusOK, res)
	}
}
