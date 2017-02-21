package frontend

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

func (f *Frontend) InitIsoApi() {
	f.MgmtApi.GET("/isos",
		func(c *gin.Context) {
			listIsos(c, f.FileRoot)
		})
	f.MgmtApi.GET("/isos/:name",
		func(c *gin.Context) {
			getIso(c, f.FileRoot, c.Param(`name`))
		})
	f.MgmtApi.POST("/isos/:name",
		func(c *gin.Context) {
			uploadIso(c, f.FileRoot, c.Param(`name`), f.DataTracker)
		})
	f.MgmtApi.DELETE("/isos/:name",
		func(c *gin.Context) {
			deleteIso(c, f.FileRoot, c.Param(`name`))
		})
}

func listIsos(c *gin.Context, fileRoot string) {
	ents, err := ioutil.ReadDir(path.Join(fileRoot, "isos"))
	if err != nil {
		c.JSON(http.StatusNotFound,
			backend.NewError("API ERROR", http.StatusNotFound, fmt.Sprintf("list: error listing isos: %v", err)))
		return
	}
	res := []string{}
	for _, ent := range ents {
		if !ent.Mode().IsRegular() {
			continue
		}
		res = append(res, ent.Name())
	}
	c.JSON(http.StatusOK, res)
}

func getIso(c *gin.Context, fileRoot, name string) {
	isoName := path.Join(fileRoot, `isos`, path.Base(name))
	c.File(isoName)
}

func reloadBootenvsForIso(dt *backend.DataTracker, name string) {
	for _, blob := range dt.FetchAll(dt.NewBootEnv()) {
		env := backend.AsBootEnv(blob)
		if env.Available || env.OS.IsoFile != name {
			continue
		}
		env.Available = true
		dt.Update(env)
	}
}

func uploadIso(c *gin.Context, fileRoot, name string, dt *backend.DataTracker) {
	if c.Request.Header.Get(`Content-Type`) != `application/octet-stream` {
		c.JSON(http.StatusUnsupportedMediaType,
			backend.NewError("API ERROR", http.StatusUnsupportedMediaType,
				fmt.Sprintf("upload: iso %s must have content-type application/octet-stream", name)))
		return
	}
	isoTmpName := path.Join(fileRoot, `isos`, fmt.Sprintf(`.%s.part`, path.Base(name)))
	isoName := path.Join(fileRoot, `isos`, path.Base(name))
	if _, err := os.Open(isoTmpName); err == nil {
		c.JSON(http.StatusConflict,
			backend.NewError("API ERROR", http.StatusConflict, fmt.Sprintf("upload: iso %s already uploading", name)))
		return
	}
	tgt, err := os.Create(isoTmpName)
	if err != nil {
		c.JSON(http.StatusConflict,
			backend.NewError("API ERROR", http.StatusConflict, fmt.Sprintf("upload: Unable to upload %s: %v", name, err)))
	}

	copied, err := io.Copy(tgt, c.Request.Body)
	if err != nil {
		os.Remove(isoTmpName)
		c.JSON(http.StatusInsufficientStorage,
			backend.NewError("API ERROR",
				http.StatusInsufficientStorage, fmt.Sprintf("upload: Failed to upload %s: %v", name, err)))
		return
	}
	if c.Request.ContentLength != 0 && copied != c.Request.ContentLength {
		os.Remove(isoTmpName)
		c.JSON(http.StatusBadRequest,
			backend.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: Failed to upload entire file %s: %d bytes expected, %d bytes recieved", name, c.Request.ContentLength, copied)))
		return
	}
	os.Remove(isoName)
	os.Rename(isoTmpName, isoName)
	res := &struct {
		Name string
		Size int64
	}{name, copied}
	go reloadBootenvsForIso(dt, name)
	c.JSON(http.StatusCreated, res)
}

func deleteIso(c *gin.Context, fileRoot, name string) {
	isoName := path.Join(fileRoot, `isos`, path.Base(name))
	if err := os.Remove(isoName); err != nil {
		c.JSON(http.StatusNotFound,
			backend.NewError("API ERROR", http.StatusNotFound, fmt.Sprintf("delete: unable to delete %s: %v", name, err)))
		return
	}
	c.JSON(http.StatusAccepted, nil)
}
