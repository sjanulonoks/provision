package frontend

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rackn/rocket-skates/backend"
)

func (f *Frontend) InitFileApi() {

	f.MgmtApi.GET(f.BasePath+"/files",
		func(c *gin.Context) {
			listFiles(c, f.FileRoot)
		})
	f.MgmtApi.GET(f.BasePath+"/files/*name",
		func(c *gin.Context) {
			getFile(c, f.FileRoot, c.Param(`name`))
		})
	f.MgmtApi.POST(f.BasePath+"/files/*name",
		func(c *gin.Context) {
			uploadFile(c, f.FileRoot, c.Param(`name`))
		})
	f.MgmtApi.DELETE(f.BasePath+"/files/*name",
		func(c *gin.Context) {
			deleteFile(c, f.FileRoot, c.Param(`name`))
		})

}

func listFiles(c *gin.Context, fileRoot string) {
	pathPart, _ := c.GetQuery("path")
	ents, err := ioutil.ReadDir(path.Join(fileRoot, "files", path.Clean(pathPart)))
	if err != nil {
		c.JSON(http.StatusNotFound,
			backend.NewError("API ERROR", http.StatusNotFound,
				fmt.Sprintf("list: error listing files: %v", err)))
		return
	}
	res := make([]string, 0, 0)
	for _, ent := range ents {
		if ent.Mode().IsRegular() {
			res = append(res, ent.Name())
		} else if ent.Mode().IsDir() {
			res = append(res, ent.Name()+"/")
		}
	}
	c.JSON(http.StatusOK, res)
}

func getFile(c *gin.Context, fileRoot, name string) {
	fileName := path.Join(fileRoot, `files`, path.Clean(name))
	c.File(fileName)
}

func uploadFile(c *gin.Context, fileRoot, name string) {
	if c.Request.Header.Get(`Content-Type`) != `application/octet-stream` {
		c.JSON(http.StatusUnsupportedMediaType,
			backend.NewError("API ERROR", http.StatusUnsupportedMediaType,
				fmt.Sprintf("upload: file %s must have content-type application/octet-stream", name)))
		return
	}
	fileTmpName := path.Join(fileRoot, `files`, fmt.Sprintf(`.%s.part`, path.Clean(name)))
	fileName := path.Join(fileRoot, `files`, path.Clean(name))
	if strings.HasSuffix(fileName, "/") {
		c.JSON(http.StatusForbidden,
			backend.NewError("API ERROR", http.StatusForbidden,
				fmt.Sprintf("upload: Cannot upload a directory")))
		return
	}
	if err := os.MkdirAll(path.Dir(fileName), 0755); err != nil {
		c.JSON(http.StatusConflict,
			backend.NewError("API ERROR", http.StatusConflict,
				fmt.Sprintf("upload: unable to create directory %s", path.Clean(path.Dir(name)))))
		return
	}
	if _, err := os.Open(fileTmpName); err == nil {
		c.JSON(http.StatusConflict,
			backend.NewError("API ERROR", http.StatusConflict,
				fmt.Sprintf("upload: file %s already uploading", name)))
		return
	}
	tgt, err := os.Create(fileTmpName)
	if err != nil {
		c.JSON(http.StatusConflict,
			backend.NewError("API ERROR", http.StatusConflict,
				fmt.Sprintf("upload: Unable to upload %s: %v", name, err)))
		return
	}

	copied, err := io.Copy(tgt, c.Request.Body)
	if err != nil {
		os.Remove(fileTmpName)
		c.JSON(http.StatusInsufficientStorage,
			backend.NewError("API ERROR", http.StatusInsufficientStorage,
				fmt.Sprintf("upload: Failed to upload %s: %v", name, err)))
		return
	}
	if c.Request.ContentLength != 0 && copied != c.Request.ContentLength {
		os.Remove(fileTmpName)
		c.JSON(http.StatusBadRequest,
			backend.NewError("API ERROR", http.StatusBadRequest,
				fmt.Sprintf("upload: Failed to upload entire file %s: %d bytes expected, %d bytes recieved", name, c.Request.ContentLength, copied)))
		return
	}
	os.Remove(fileName)
	os.Rename(fileTmpName, fileName)
	res := &struct {
		Name string
		Size int64
	}{name, copied}
	c.JSON(http.StatusCreated, res)
}

func deleteFile(c *gin.Context, fileRoot, name string) {
	fileName := path.Join(fileRoot, `files`, path.Clean(name))
	if fileName == path.Join(fileRoot, `files`) {
		c.JSON(http.StatusForbidden,
			backend.NewError("API ERROR", http.StatusForbidden,
				fmt.Sprintf("delete: Not allowed to remove files dir")))
		return
	}
	if err := os.Remove(fileName); err != nil {
		c.JSON(http.StatusNotFound,
			backend.NewError("API ERROR", http.StatusNotFound,
				fmt.Sprintf("delete: unable to delete %s: %v", name, err)))
		return
	}
	c.JSON(http.StatusAccepted, nil)
}
