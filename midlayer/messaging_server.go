package midlayer

import (
	"net/http"
	"strings"
	"time"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

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

func publishHandler(c *gin.Context, pc *PluginClient) {
	var event models.Event
	if !assureDecode(c, &event) {
		return
	}
	resp := models.Error{Code: http.StatusOK}
	if err := pc.pc.Request().PublishEvent(&event); err != nil {
		resp.Code = 409
		resp.AddError(err)
	}
	c.JSON(resp.Code, resp)
}

func logHandler(c *gin.Context, pc *PluginClient) {
	var line logger.Line
	if !assureDecode(c, &line) {
		return
	}
	if line.Level == logger.Fatal || line.Level == logger.Panic {
		line.Level = logger.Error
	}
	pc.AddLine(&line)
	c.JSON(204, nil)
}

func newGinServer(bl logger.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	mgmtApi := gin.New()
	mgmtApi.Use(func(c *gin.Context) {
		l := bl.Fork()
		if logLevel := c.GetHeader("X-Log-Request"); logLevel != "" {
			lvl, err := logger.ParseLevel(logLevel)
			if err != nil {
				l.Errorf("Invalid requested log level %s", logLevel)
			} else {
				l = l.Trace(lvl)
			}
		}
		if logToken := c.GetHeader("X-Log-Token"); logToken != "" {
			l.Errorf("Log token: %s", logToken)
		}
		c.Set("logger", l)
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		c.Next()
		latency := time.Now().Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		if raw != "" {
			path = path + "?" + raw
		}
		l.Debugf("API: st: %d lt: %13v ip: %15s m: %s %s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	})
	mgmtApi.Use(gin.Recovery())

	return mgmtApi
}

func (pc *PluginClient) pluginServer(commDir string) {
	pc.Tracef("pluginServer: Starting com server: %s(%s)\n", pc.plugin, commDir)

	gc := newGinServer(pc.NoPublish())
	apiGroup := gc.Group("/api-server-plugin/v3")

	apiGroup.POST("/publish", func(c *gin.Context) { publishHandler(c, pc) })
	apiGroup.POST("/log", func(c *gin.Context) { logHandler(c, pc) })
	// apiGroup.POST("/object", func(c *gin.Context) { objectHandler(c, pc) })

	go func() {
		if err := gc.RunUnix(commDir); err != nil {
			pc.Errorf("pluginServer: Finished (error) com server: %s(%s): %v\n", pc.plugin, commDir, err)
		} else {
			pc.Tracef("pluginServer: Finished com server: %s(%s)\n", pc.plugin, commDir)
		}
	}()
}
