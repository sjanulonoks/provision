package frontend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/embedded"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
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
	FetchAll(store.KeySaver) []store.KeySaver
	Filter(store.KeySaver, ...index.Filter) ([]store.KeySaver, error)

	NewBootEnv() *backend.BootEnv
	NewMachine() *backend.Machine
	NewTemplate() *backend.Template
	NewLease() *backend.Lease
	NewReservation() *backend.Reservation
	NewSubnet() *backend.Subnet
	NewUser() *backend.User
	NewProfile() *backend.Profile

	Pref(string) (string, error)
	Prefs() map[string]string
	SetPrefs(map[string]string) error

	GetInterfaces() ([]*backend.Interface, error)

	GetToken(string) (*backend.DrpCustomClaims, error)
	NewToken(string, int, string, string, string) (string, error)
}

type Sanitizable interface {
	Sanitize()
}

type Indexable interface {
	Indexes() map[string]index.Maker
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
	gin.SetMode(gin.ReleaseMode)

	if authSource == nil {
		authSource = NewDefaultAuthSource(dt)
	}

	userAuth := func() gin.HandlerFunc {
		return func(c *gin.Context) {
			authHeader := c.Request.Header.Get("Authorization")
			if len(authHeader) == 0 {
				logger.Printf("No authentication header")
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			hdrParts := strings.SplitN(authHeader, " ", 2)
			if len(hdrParts) != 2 || (hdrParts[0] != "Basic" && hdrParts[0] != "Bearer") {
				logger.Printf("Bad auth header: %s", authHeader)
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			if hdrParts[0] == "Basic" {
				hdr, err := base64.StdEncoding.DecodeString(hdrParts[1])
				if err != nil {
					logger.Printf("Malformed basic auth string: %s", hdrParts[1])
					c.Header("WWW-Authenticate", "dr-provision")
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
				userpass := bytes.SplitN(hdr, []byte(`:`), 2)
				if len(userpass) != 2 {
					logger.Printf("Malformed basic auth string: %s", hdrParts[1])
					c.Header("WWW-Authenticate", "dr-provision")
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
				t := backend.NewClaim(string(userpass[0]), 30).Add("*", "*", "*")
				c.Set("DRP-CLAIM", t)
			} else if hdrParts[0] == "Bearer" {
				t, err := dt.GetToken(string(hdrParts[1]))
				if err != nil {
					logger.Printf("No DRP authentication token")
					c.Header("WWW-Authenticate", "dr-provision")
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.Set("DRP-CLAIM", t)
			}

			c.Next()
		}
	}

	mgmtApi := gin.Default()

	apiGroup := mgmtApi.Group("/api/v3")
	apiGroup.Use(userAuth())

	me = &Frontend{Logger: logger, FileRoot: fileRoot, MgmtApi: mgmtApi, ApiGroup: apiGroup, dt: dt}

	me.InitBootEnvApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()
	me.InitProfileApi()
	me.InitLeaseApi()
	me.InitReservationApi()
	me.InitSubnetApi()
	me.InitUserApi()
	me.InitInterfaceApi()
	me.InitPrefApi()

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
	if !drpClaim.Match(scope, action, specific) {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	return true
}

func assureDecode(c *gin.Context, val interface{}) bool {
	if !assureContentType(c, "application/json") {
		return false
	}
	marshalErr := c.Bind(&val)
	if marshalErr == nil {
		return true
	}
	err := &backend.Error{Type: "API_ERROR", Code: http.StatusBadRequest}
	err.Merge(marshalErr)
	c.JSON(err.Code, err)
	return false
}

func (f *Frontend) processFilters(ref store.KeySaver, params map[string][]string) []index.Filter {
	filters := []index.Filter{}

	var indexes map[string]index.Maker
	if indexer, ok := ref.(Indexable); ok {
		indexes = indexer.Indexes()
	} else {
		indexes = map[string]index.Maker{}
	}

	for k, vs := range params {
		if k == "offset" || k == "limit" || k == "sort" {
			continue
		}

		if maker, ok := indexes[k]; ok {
			filters = append(filters, index.Sort(maker))
			subfilters := []index.Filter{}
			for _, v := range vs {
				subfilters = append(subfilters, index.Eq(v))
			}
			filters = append(filters, index.Any(subfilters...))
		} else {
			f.Logger.Printf("GREG: ERROR: filter not found: %s\n", k)
		}
	}

	if vs, ok := params["sort"]; ok {
		for _, piece := range vs {
			if maker, ok := indexes[piece]; ok {
				filters = append(filters, index.Sort(maker))
			} else {
				f.Logger.Printf("GREG: ERROR: not sortable: %s\n", piece)
			}
		}
	} else {
		filters = append(filters, index.Native())
	}

	// offset and limit must be last
	if vs, ok := params["offset"]; ok {
		num, err := strconv.Atoi(vs[0])
		if err == nil {
			filters = append(filters, index.Offset(num))
		} else {
			f.Logger.Printf("GREG: ERROR: offset not valid: %v\n", err)
		}
	}
	if vs, ok := params["limit"]; ok {
		num, err := strconv.Atoi(vs[0])
		if err == nil {
			filters = append(filters, index.Limit(num))
		} else {
			f.Logger.Printf("GREG: ERROR: limit not valid: %v\n", err)
		}
	}

	return filters
}

func (f *Frontend) List(c *gin.Context, ref store.KeySaver) {
	if !assureAuth(c, f.Logger, ref.Prefix(), "list", "") {
		return
	}
	filters := f.processFilters(ref, c.Request.URL.Query())
	arr, err := f.dt.Filter(ref, filters...)
	if err != nil {
		res := &backend.Error{
			Code:  http.StatusNotAcceptable,
			Type:  "API_ERROR",
			Model: ref.Prefix(),
		}
		res.Merge(err)
		c.JSON(res.Code, res)
		return
	}
	for _, res := range arr {
		s, ok := res.(Sanitizable)
		if ok {
			s.Sanitize()
		}
	}
	c.JSON(http.StatusOK, arr)
}

func (f *Frontend) Fetch(c *gin.Context, ref store.KeySaver, key string) {
	res, ok := f.dt.FetchOne(ref, key)
	if ok {
		// TODO: This should really be done before the fetch - it may have issue with HexAddr-based things.
		if !assureAuth(c, f.Logger, ref.Prefix(), "get", res.Key()) {
			return
		}
		s, ok := res.(Sanitizable)
		if ok {
			s.Sanitize()
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
		s, ok := res.(Sanitizable)
		if ok {
			s.Sanitize()
		}
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
		s, ok := res.(Sanitizable)
		if ok {
			s.Sanitize()
		}
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
		s, ok := newThing.(Sanitizable)
		if ok {
			s.Sanitize()
		}
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
		s, ok := res.(Sanitizable)
		if ok {
			s.Sanitize()
		}
		c.JSON(http.StatusOK, res)
	}
}
