package frontend

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (f *Frontend) InitStatsApi() {
	f.ApiGroup.GET("/stats/publishers",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "stats", "list", "") {
				return
			}
			pubs := f.pubs.List()
			c.JSON(http.StatusOK, pubs)
		})
}
