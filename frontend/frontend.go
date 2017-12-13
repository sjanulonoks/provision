package frontend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	melody "gopkg.in/olahol/melody.v1"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/embedded"
	"github.com/digitalrebar/provision/midlayer"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ErrorResponse is returned whenever an error occurs
// swagger:response
type ErrorResponse struct {
	//in: body
	Body models.Error
}

// NoContentResponse is returned for deletes and auth errors
// swagger:response
type NoContentResponse struct {
	//description: Nothing
}

type Sanitizable interface {
	Sanitize() models.Model
}

type Lockable interface {
	Locks(string) []string
}

type Frontend struct {
	Logger     *log.Logger
	FileRoot   string
	MgmtApi    *gin.Engine
	ApiGroup   *gin.RouterGroup
	dt         *backend.DataTracker
	pc         *midlayer.PluginController
	authSource AuthSource
	pubs       *backend.Publishers
	melody     *melody.Melody
	ApiPort    int
	ProvPort   int
	TftpPort   int
	DhcpPort   int
	PxePort    int
	NoDhcp     bool
	NoTftp     bool
	NoProv     bool
	NoPxe      bool
	SaasDir    string
}

type AuthSource interface {
	GetUser(username string) *backend.User
}

type DefaultAuthSource struct {
	dt *backend.DataTracker
}

func (d DefaultAuthSource) GetUser(username string) *backend.User {
	objs, unlocker := d.dt.LockEnts("users")
	defer unlocker()
	u := objs("users").Find(username)
	if u != nil {
		return u.(*backend.User)
	}
	return nil
}

func NewDefaultAuthSource(dt *backend.DataTracker) (das AuthSource) {
	das = DefaultAuthSource{dt: dt}
	return
}

func (f *Frontend) makeParamEndpoints(obj backend.Paramer, idKey string) (
	getAll, getOne, patchThem, setThem, setOne, deleteOne func(c *gin.Context)) {
	trimmer := func(s string) string {
		return strings.TrimLeft(s, `/`)
	}
	aggregator := func(c *gin.Context) bool {
		return c.Query("aggregate") == "true"
	}
	pFetch := func(obj backend.Paramer, id string, aggregate bool) (
		bool,
		map[string]interface{},
	) {
		d, unlocker := f.dt.LockEnts(obj.(Lockable).Locks("get")...)
		defer unlocker()
		ref := d(obj.Prefix()).Find(id)
		if ref != nil {
			return true, ref.(backend.Paramer).GetParams(d, aggregate)
		}
		return false, nil
	}
	pFetchOne := func(obj backend.Paramer, id, key string, aggregate bool) (bool, interface{}) {
		d, unlocker := f.dt.LockEnts(obj.(Lockable).Locks("get")...)
		defer unlocker()
		ref := d(obj.Prefix()).Find(id)
		if ref == nil {
			return false, nil
		}
		v, _ := ref.(backend.Paramer).GetParam(d, key, aggregate)
		return true, v
	}
	pSet := func(c *gin.Context,
		key, line string,
		munger func(backend.Paramer,
			map[string]interface{}) (map[string]interface{},
			interface{},
			*models.Error)) (interface{}, *models.Error) {
		d, unlocker := f.dt.LockEnts(obj.(Lockable).Locks("update")...)
		defer unlocker()
		thing := d(obj.Prefix()).Find(key)
		err := &models.Error{
			Code:  http.StatusNotFound,
			Type:  c.Request.Method,
			Model: obj.Prefix(),
			Key:   key,
		}
		if thing == nil {
			err.Errorf("Not Found")
			return nil, err
		}
		ref := thing.(backend.Paramer)
		params, res, err := munger(ref, ref.GetParams(d, false))
		if err != nil {
			return res, err
		}
		if setErr := ref.SetParams(d, params); setErr != nil {
			return res, setErr.(*models.Error)
		}
		return res, nil
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
	return /* getAll */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			found, params := pFetch(obj, id, aggregator(c))
			if item404(c, found, id, "Params") {
				return
			}
			c.JSON(http.StatusOK, params)
		},
		/* getOne */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			found, val := pFetchOne(obj, id, trimmer(c.Param("key")), aggregator(c))
			if item404(c, found, id, "Param") {
				return
			}
			c.JSON(http.StatusOK, val)
		},
		/* patchThem */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			var patch jsonpatch2.Patch
			if !assureDecode(c, &patch) {
				return
			}
			res, err := pSet(c, id, "Params",
				func(m backend.Paramer,
					params map[string]interface{}) (map[string]interface{},
					interface{},
					*models.Error) {
					var val map[string]interface{}
					res := &models.Error{
						Code:  http.StatusConflict,
						Type:  c.Request.Method,
						Model: obj.Prefix(),
						Key:   id,
					}
					buf, err := json.Marshal(params)
					if err != nil {
						res.AddError(err)
						return val, nil, res
					}
					patched, err, loc := patch.Apply(buf)
					if err != nil {
						res.Errorf("Patch failed to apply at line %d", loc)
						res.AddError(err)
						return val, nil, res
					}
					if err := json.Unmarshal(patched, &val); err != nil {
						res.AddError(err)
						return val, nil, res
					}
					return val, val, nil
				})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, res)
			}
		},
		/* setThem */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			var replacement map[string]interface{}
			if !assureDecode(c, &replacement) {
				return
			}
			res, err := pSet(c, id, "Params",
				func(m backend.Paramer,
					params map[string]interface{}) (map[string]interface{}, interface{},
					*models.Error) {
					return replacement, replacement, nil
				})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, res)
			}
		},
		/* setOne */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			var replacement interface{}
			if !assureDecode(c, &replacement) {
				return
			}
			res, err := pSet(c, id, "Params",
				func(m backend.Paramer,
					params map[string]interface{}) (map[string]interface{}, interface{},
					*models.Error) {
					params[trimmer(c.Param("key"))] = replacement
					return params, replacement, nil
				})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, res)
			}
		},
		/* deleteOne */ func(c *gin.Context) {
			id := c.Param(idKey)
			if !f.assureAuth(c, obj.Prefix(), "get", id) {
				return
			}
			res, err := pSet(c, id, "Params",
				func(m backend.Paramer,
					params map[string]interface{}) (map[string]interface{}, interface{},
					*models.Error) {
					k := trimmer(c.Param("key"))
					if _, ok := params[k]; ok {
						res := params[k]
						delete(params, k)
						return params, res, nil
					} else {
						return params, nil, &models.Error{
							Code:  http.StatusNotFound,
							Type:  c.Request.Method,
							Model: "params",
							Key:   k,
						}
					}
				})
			if err != nil {
				c.JSON(err.Code, err)
			} else {
				c.JSON(http.StatusOK, res)
			}
		}
}

func (fe *Frontend) userAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if len(authHeader) == 0 {
			authHeader = c.Query("token")
			if len(authHeader) == 0 {
				fe.Logger.Printf("No authentication header or token")
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			} else {
				if strings.Contains(authHeader, ":") {
					authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(authHeader))
				} else {
					authHeader = "Bearer " + authHeader
				}
			}
		}
		hdrParts := strings.SplitN(authHeader, " ", 2)
		if len(hdrParts) != 2 || (hdrParts[0] != "Basic" && hdrParts[0] != "Bearer") {
			fe.Logger.Printf("Bad auth header: %s", authHeader)
			c.Header("WWW-Authenticate", "dr-provision")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if hdrParts[0] == "Basic" {
			hdr, err := base64.StdEncoding.DecodeString(hdrParts[1])
			if err != nil {
				fe.Logger.Printf("Malformed basic auth string: %s", hdrParts[1])
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			userpass := bytes.SplitN(hdr, []byte(`:`), 2)
			if len(userpass) != 2 {
				fe.Logger.Printf("Malformed basic auth string: %s", hdrParts[1])
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			user := fe.authSource.GetUser(string(userpass[0]))
			if user == nil {
				fe.Logger.Printf("No such user: %s", string(userpass[0]))
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if !user.CheckPassword(string(userpass[1])) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			t := backend.NewClaim(string(userpass[0]), string(userpass[0]), 30).Add("*", "*", "*")
			c.Set("DRP-CLAIM", t)
		} else if hdrParts[0] == "Bearer" {
			t, err := fe.dt.GetToken(string(hdrParts[1]))
			if err != nil {
				fe.Logger.Printf("No DRP authentication token")
				c.Header("WWW-Authenticate", "dr-provision")
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Set("DRP-CLAIM", t)
		}
		c.Next()
	}
}

func NewFrontend(
	dt *backend.DataTracker,
	logger *log.Logger,
	address string,
	apiport, provport, dhcpport, pxeport int,
	fileRoot, devUI, UIUrl string,
	authSource AuthSource,
	pubs *backend.Publishers,
	drpid string,
	pc *midlayer.PluginController,
	noDhcp, noTftp, noProv, noPxe bool,
	saasDir string) (me *Frontend) {
	me = &Frontend{
		Logger:     logger,
		FileRoot:   fileRoot,
		dt:         dt,
		pubs:       pubs,
		pc:         pc,
		ApiPort:    apiport,
		ProvPort:   provport,
		DhcpPort:   dhcpport,
		PxePort:    pxeport,
		NoDhcp:     noDhcp,
		NoTftp:     noTftp,
		NoProv:     noProv,
		NoPxe:      noPxe,
		SaasDir:    saasDir,
		authSource: authSource,
	}
	gin.SetMode(gin.ReleaseMode)

	if me.authSource == nil {
		me.authSource = NewDefaultAuthSource(dt)
	}

	mgmtApi := gin.New()
	if dt.DebugLevel("debugFrontend") > 0 {
		mgmtApi.Use(gin.Logger())
	}
	mgmtApi.Use(gin.Recovery())

	// CORS Support
	mgmtApi.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"},
		AllowHeaders: []string{
			"Origin",
			"X-Requested-With",
			"Content-Type",
			"Cookie",
			"Authorization",
			"WWW-Authenticate",
			"X-Return-Attributes",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"WWW-Authenticate",
			"Set-Cookie",
			"Access-Control-Allow-Headers",
			"Access-Control-Allow-Credentials",
			"Access-Control-Allow-Origin",
			"X-Return-Attributes",
			"X-DRP-LIST-COUNT",
			"X-DRP-LIST-TOTAL-COUNT",
		},
	}))

	mgmtApi.Use(location.Default())
	me.MgmtApi = mgmtApi

	apiGroup := mgmtApi.Group("/api/v3")
	apiGroup.Use(me.userAuth())
	me.ApiGroup = apiGroup

	me.InitIndexApi()
	me.InitWebSocket()
	me.InitBootEnvApi()
	me.InitStageApi()
	me.InitIsoApi()
	me.InitFileApi()
	me.InitTemplateApi()
	me.InitMachineApi()
	me.InitProfileApi()
	me.InitLeaseApi()
	me.InitReservationApi()
	me.InitSubnetApi()
	me.InitUserApi(drpid)
	me.InitInterfaceApi()
	me.InitPrefApi()
	me.InitParamApi()
	me.InitInfoApi(drpid)
	me.InitPluginApi()
	me.InitPluginProviderApi()
	me.InitTaskApi()
	me.InitJobApi()
	me.InitEventApi()
	me.InitContentApi()

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

	// Optionally add a local dev-ui
	if len(devUI) != 0 {
		logger.Printf("DEV: Running UI from %s\n", devUI)
		mgmtApi.Static("/dev-ui", devUI)
	}

	// UI points to the cloud
	mgmtApi.GET("/ui", func(c *gin.Context) {
		incomingUrl := location.Get(c)

		url := fmt.Sprintf("%s/#/e/%s", UIUrl, incomingUrl.Host)
		c.Redirect(http.StatusMovedPermanently, url)
	})

	// root path, forward to UI
	mgmtApi.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/ui/")
	})

	pubs.Add(me)

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
	err := &models.Error{Type: c.Request.Method, Code: http.StatusBadRequest}
	err.Errorf("Invalid content type: %s", c.ContentType())
	c.JSON(err.Code, err)
	return false
}

func (f *Frontend) getAuthUser(c *gin.Context) string {
	obj, ok := c.Get("DRP-CLAIM")
	if !ok {
		return ""
	}
	drpClaim, ok := obj.(*backend.DrpCustomClaims)
	if !ok {
		return ""
	}
	return drpClaim.Id
}

//
// THIS MUST NOT BE CALLED UNDER LOCKS!
//
func (f *Frontend) assureAuth(c *gin.Context, scope, action, specific string) bool {
	obj, ok := c.Get("DRP-CLAIM")
	if !ok {
		f.Logger.Printf("Request with no claims\n")
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	drpClaim, ok := obj.(*backend.DrpCustomClaims)
	if !ok {
		f.Logger.Printf("Request with bad claims\n")
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}
	if !drpClaim.Match(scope, action, specific) {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	userSecret := ""
	grantorSecret := ""
	machineSecret := ""

	if drpClaim.HasUserId() {
		ref := &backend.User{}
		func() {
			d, unlocker := f.dt.LockEnts(ref.Locks("get")...)
			defer unlocker()
			if obj := d("users").Find(drpClaim.UserId()); obj != nil {
				userSecret = backend.AsUser(obj).Secret
			}
		}()
	}
	if drpClaim.HasGrantorId() {
		if drpClaim.GrantorId() != "system" {
			ref := &backend.User{}
			func() {
				d, unlocker := f.dt.LockEnts(ref.Locks("get")...)
				defer unlocker()
				if obj := d("users").Find(drpClaim.UserId()); obj != nil {
					grantorSecret = backend.AsUser(obj).Secret
				}
			}()
		} else {
			prefs := f.dt.Prefs()
			if ss, ok := prefs["systemGrantorSecret"]; ok {
				grantorSecret = ss
			}
		}
	}
	if drpClaim.HasMachineUuid() {
		ref := &backend.Machine{}
		func() {
			d, unlocker := f.dt.LockEnts(ref.Locks("get")...)
			defer unlocker()
			if obj := d("machines").Find(drpClaim.MachineUuid()); obj != nil {
				machineSecret = backend.AsMachine(obj).Secret
			}
		}()
		if machineSecret == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return false
		}
	}

	if !drpClaim.ValidateSecrets(grantorSecret, userSecret, machineSecret) {
		c.AbortWithStatus(http.StatusForbidden)
		return false
	}

	return true
}

func assureDecode(c *gin.Context, val interface{}) bool {
	if !assureContentType(c, "application/json") {
		return false
	}
	if c.Request.ContentLength == 0 {
		val = nil
		return true
	}
	marshalErr := binding.JSON.Bind(c.Request, &val)
	if marshalErr == nil {
		return true
	}
	err := &models.Error{Type: c.Request.Method, Code: http.StatusBadRequest}
	err.AddError(marshalErr)
	c.JSON(err.Code, err)
	return false
}

// This processes the value into a function, if function not specifed, assume Eq.
// Supported Forms:
//
//   Eq(value)
//   Lt(value)
//   Lte(value)
//   Gt(value)
//   Gte(value)
//   Ne(value)
//   Between(valueLower, valueHigher)
//   Except(valueLower, valueHigher)
//
func convertValueToFilter(v string) (index.Filter, error) {
	args := strings.SplitN(v, "(", 2)
	switch args[0] {
	case "Eq":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Eq(subargs[0]), nil
	case "Lt":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Lt(subargs[0]), nil
	case "Lte":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Lte(subargs[0]), nil
	case "Gt":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Gt(subargs[0]), nil
	case "Gte":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Gte(subargs[0]), nil
	case "Ne":
		subargs := strings.SplitN(args[1], ")", 2)
		return index.Ne(subargs[0]), nil
	case "Between":
		subargs := strings.SplitN(args[1], ")", 2)
		parts := strings.Split(subargs[0], ",")
		return index.Between(parts[0], parts[1]), nil
	case "Except":
		subargs := strings.SplitN(args[1], ")", 2)
		parts := strings.Split(subargs[0], ",")
		return index.Except(parts[0], parts[1]), nil
	default:
		return index.Eq(v), nil
	}
	return nil, fmt.Errorf("Should never get here")
}

type dynParameter interface {
	ParameterMaker(backend.Stores, string) (index.Maker, error)
}

func (f *Frontend) processFilters(d backend.Stores, ref models.Model, params map[string][]string) ([]index.Filter, error) {
	filters := []index.Filter{}
	var err error
	var indexes map[string]index.Maker
	if indexer, ok := ref.(index.Indexer); ok {
		indexes = indexer.Indexes()
	} else {
		indexes = map[string]index.Maker{}
	}

	for k, vs := range params {
		if k == "offset" || k == "limit" || k == "sort" || k == "reverse" {
			continue
		}
		maker, ok := indexes[k]
		pMaker, found := ref.(dynParameter)
		if !ok {
			if !found {
				return nil, fmt.Errorf("Filter not found: %s", k)
			}
			maker, err = pMaker.ParameterMaker(d, k)
			if err != nil {
				return nil, err
			}
			ok = true
		}
		if ok {
			filters = append(filters, index.Sort(maker))
			subfilters := []index.Filter{}
			for _, v := range vs {
				f, err := convertValueToFilter(v)
				if err != nil {
					return nil, err
				}
				subfilters = append(subfilters, f)
			}
			filters = append(filters, index.Any(subfilters...))
		}
	}

	if vs, ok := params["sort"]; ok {
		for _, piece := range vs {
			if maker, ok := indexes[piece]; ok {
				filters = append(filters, index.Sort(maker))
			} else {
				return nil, fmt.Errorf("Not sortable: %s", piece)
			}
		}
	} else {
		filters = append(filters, index.Native())
	}

	if _, ok := params["reverse"]; ok {
		filters = append(filters, index.Reverse())
	}

	// offset and limit must be last
	if vs, ok := params["offset"]; ok {
		num, err := strconv.Atoi(vs[0])
		if err == nil {
			filters = append(filters, index.Offset(num))
		} else {
			return nil, fmt.Errorf("Offset not valid: %v", err)
		}
	}
	if vs, ok := params["limit"]; ok {
		num, err := strconv.Atoi(vs[0])
		if err == nil {
			filters = append(filters, index.Limit(num))
		} else {
			return nil, fmt.Errorf("Limit not valid: %v", err)
		}
	}

	return filters, nil
}

func jsonError(c *gin.Context, err error, code int, base string) {
	if ne, ok := err.(*models.Error); ok {
		c.JSON(ne.Code, ne)
	} else {
		res := &models.Error{
			Type:  c.Request.Method,
			Code:  code,
			Model: base,
		}
		res.AddError(err)
		c.JSON(res.Code, res)
	}
}

func (f *Frontend) list(c *gin.Context, ref store.KeySaver, statsOnly bool) {
	if !f.assureAuth(c, ref.Prefix(), "list", "") {
		return
	}
	res := &models.Error{
		Code:  http.StatusNotAcceptable,
		Type:  c.Request.Method,
		Model: ref.Prefix(),
	}
	var err error
	arr := []models.Model{}
	func() {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("get")...)
		defer unlocker()
		var filters []index.Filter
		filters, err = f.processFilters(d, ref, c.Request.URL.Query())
		if err != nil {
			res.AddError(err)
			return
		}

		mainIndex := &d(ref.Prefix()).Index
		c.Header("X-DRP-LIST-TOTAL-COUNT", fmt.Sprintf("%d", mainIndex.Count()))

		idx, err := index.All(filters...)(mainIndex)
		if err != nil {
			res.AddError(err)
			return
		}

		c.Header("X-DRP-LIST-COUNT", fmt.Sprintf("%d", idx.Count()))
		if statsOnly {
			return
		}

		items := idx.Items()
		for i, item := range items {
			arr = append(arr, models.Clone(item))
			f, ok := arr[i].(models.Filler)
			if ok {
				f.Fill()
			}
			s, ok := arr[i].(Sanitizable)
			if ok {
				arr[i] = s.Sanitize()
			}
		}
	}()

	if res.ContainsError() {
		c.JSON(res.Code, res)
		return
	}

	if statsOnly {
		c.Status(http.StatusOK)
	} else {
		c.JSON(http.StatusOK, arr)
	}
}

func (f *Frontend) ListStats(c *gin.Context, ref store.KeySaver) {
	f.list(c, ref, true)
}

// XXX: Auth enforce may need to limit return values based up access to get - one day.
func (f *Frontend) List(c *gin.Context, ref store.KeySaver) {
	f.list(c, ref, false)
}

func (f *Frontend) Exists(c *gin.Context, ref store.KeySaver, key string) {
	prefix := ref.Prefix()
	var found bool
	func() {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("get")...)
		defer unlocker()
		objs := d(prefix)
		idxer, ok := ref.(index.Indexer)
		if ok {
			for idxName, idx := range idxer.Indexes() {
				idxKey := strings.TrimPrefix(key, idxName+":")
				if key == idxKey {
					continue
				}
				if !idx.Unique {
					break
				}
				items, err := index.All(index.Sort(idx))(&objs.Index)
				if err == nil {
					found = items.Find(idxKey) != nil
				}
				break
			}
		}
		if !found {
			found = objs.Find(key) != nil
		}
	}()
	if found {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusNotFound)
	}
}

func (f *Frontend) Fetch(c *gin.Context, ref store.KeySaver, key string) {
	prefix := ref.Prefix()
	var err error
	var res models.Model
	func() {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("get")...)
		defer unlocker()
		objs := d(prefix)
		idxer, ok := ref.(index.Indexer)
		found := false
		if ok {
			for idxName, idx := range idxer.Indexes() {
				idxKey := strings.TrimPrefix(key, idxName+":")
				if key == idxKey {
					continue
				}
				found = true
				ref = nil
				if !idx.Unique {
					break
				}
				items, err := index.All(index.Sort(idx))(&objs.Index)
				if err == nil {
					res = models.Clone(items.Find(idxKey))
				}
				break
			}
		}
		if !found {
			res = models.Clone(objs.Find(key))
		}
	}()
	if res != nil {
		aref, _ := res.(backend.AuthSaver)
		if !f.assureAuth(c, prefix, "get", aref.AuthKey()) {
			return
		}
		s, ok := res.(Sanitizable)
		if ok {
			res = s.Sanitize()
		}
		c.JSON(http.StatusOK, res)
	} else {
		rerr := &models.Error{
			Code:  http.StatusNotFound,
			Type:  c.Request.Method,
			Model: prefix,
			Key:   key,
		}
		rerr.Errorf("Not Found")
		if err != nil {
			rerr.AddError(err)
		}
		c.JSON(rerr.Code, rerr)
	}
}

func (f *Frontend) Create(c *gin.Context, val store.KeySaver) {
	if !assureDecode(c, val) {
		return
	}
	if !f.assureAuth(c, val.Prefix(), "create", "") {
		return
	}
	var err error
	var res models.Model
	func() {
		d, unlocker := f.dt.LockEnts(val.(Lockable).Locks("create")...)
		defer unlocker()
		_, err = f.dt.Create(d, val)
		if err == nil {
			res = models.Clone(val)
		}
	}()
	if err != nil {
		jsonError(c, err, http.StatusBadRequest, "")
	} else {
		s, ok := res.(Sanitizable)
		if ok {
			res = s.Sanitize()
		}
		c.JSON(http.StatusCreated, res)
	}
}

func (f *Frontend) Patch(c *gin.Context, ref store.KeySaver, key string) {
	patch := make(jsonpatch2.Patch, 0)
	if !assureDecode(c, &patch) {
		return
	}
	var err error
	var tref models.Model
	authKey := ""

	func() {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("update")...)
		defer unlocker()

		tref = d(ref.Prefix()).Find(key)
		if tref != nil {
			authKey = tref.(backend.AuthSaver).AuthKey()
		}
	}()

	if authKey != "" && !f.assureAuth(c, ref.Prefix(), "patch", authKey) {
		return
	}

	var res models.Model
	res, err = func() (models.Model, error) {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("update")...)
		defer unlocker()
		// This will fail with notfound as well.
		a, b := f.dt.Patch(d, ref, key, patch)
		return models.Clone(a), b
	}()
	if err == nil {
		s, ok := res.(Sanitizable)
		if ok {
			res = s.Sanitize()
		}
		c.JSON(http.StatusOK, res)
		return
	}
	jsonError(c, err, http.StatusBadRequest, "")
}

func (f *Frontend) Update(c *gin.Context, ref store.KeySaver, key string) {
	if !assureDecode(c, ref) {
		return
	}
	if ref.Key() != key {
		err := &models.Error{
			Code:  http.StatusBadRequest,
			Type:  c.Request.Method,
			Model: ref.Prefix(),
			Key:   key,
		}
		err.Errorf("Key change from %s to %s not allowed", key, ref.Key())
		c.JSON(err.Code, err)
		return
	}
	var err error
	authKey := ""
	func() {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("update")...)
		defer unlocker()

		tref := d(ref.Prefix()).Find(ref.Key())
		if tref != nil {
			authKey = tref.(backend.AuthSaver).AuthKey()
		}
	}()
	if !f.assureAuth(c, ref.Prefix(), "update", authKey) {
		return
	}
	var res models.Model
	res, err = func() (models.Model, error) {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("update")...)
		defer unlocker()
		_, b := f.dt.Update(d, ref)
		return models.Clone(ref), b
	}()
	if err == nil {
		s, ok := ref.(Sanitizable)
		if ok {
			res = s.Sanitize()
		}
		c.JSON(http.StatusOK, res)
		return
	}
	jsonError(c, err, http.StatusBadRequest, "")
}

func (f *Frontend) Remove(c *gin.Context, ref store.KeySaver, key string) {
	var err error
	var res models.Model

	res, err = func() (models.Model, error) {
		d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("delete")...)
		defer unlocker()
		a := models.Clone(d(ref.Prefix()).Find(key))
		if a == nil {
			b := &models.Error{
				Type:     "DELETE",
				Code:     http.StatusNotFound,
				Key:      key,
				Model:    ref.Prefix(),
				Messages: []string{"Not Found"},
			}
			return a, b
		}
		return a, nil
	}()

	if res != nil {
		aref := res.(backend.AuthSaver)
		if !f.assureAuth(c, ref.Prefix(), "delete", aref.AuthKey()) {
			return
		}

		func() {
			d, unlocker := f.dt.LockEnts(ref.(Lockable).Locks("delete")...)
			defer unlocker()
			_, err = f.dt.Remove(d, res)
		}()
	}

	if err != nil {
		jsonError(c, err, http.StatusNotFound, "")
	} else {
		s, ok := res.(Sanitizable)
		if ok {
			res = s.Sanitize()
		}
		c.JSON(http.StatusOK, res)
	}
}
