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

type IsoPaths []string

// IsosResponse returned on a successful GET of isos
// swagger:response
type IsosResponse struct {
	// in: body
	Body IsoPaths
}

// XXX: One day resolve the binary blob appropriately:
// {
//   "name": "BinaryData",
//   "in": "body",
//   "required": true,
//   "schema": {
//     "type": "string",
//     "format": "byte"
//   }
// }
//

// IsoResponse returned on a successful GET of an iso
// swagger:response
type IsoResponse struct {
	// in: body
	Body interface{}
}

type IsoInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// IsoInfoResponse returned on a successful upload of an iso
// swagger:response
type IsoInfoResponse struct {
	// in: body
	Body *IsoInfo
}

// swagger:parameters uploadIso getIso deleteIso
type IsoPathPathParameter struct {
	// in: path
	Path string `json:"path"`
}

// IsoData body of the upload
// swagger:parameters uploadIso
type IsoData struct {
	// in: body
	Body interface{}
}

func (f *Frontend) InitIsoApi() {
	// swagger:route GET /isos Isos listIsos
	//
	// Lists isos in isos directory
	//
	// Lists the isos in a directory under /isos.
	//
	//     Responses:
	//       200: IsosResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/isos",
		func(c *gin.Context) {
			ents, err := ioutil.ReadDir(path.Join(f.FileRoot, "isos"))
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
		})
	// swagger:route GET /isos/{path} Isos getIso
	//
	// Get a specific Iso with {path}
	//
	// Get a specific iso specified by {path} under isos.
	//
	//     Produces:
	//       application/octet-stream
	//       application/json
	//
	//     Responses:
	//       200: IsoResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/isos/:name",
		func(c *gin.Context) {
			isoName := path.Join(f.FileRoot, `isos`, path.Base(c.Param(`name`)))
			c.File(isoName)
		})
	// swagger:route POST /isos/{path} Isos uploadIso
	//
	// Upload an iso to a specific {path} in the tree under isos.
	//
	// The iso will be uploaded to the {path} in /isos.  The {path} will be created.
	//
	//     Consumes:
	//       application/octet-stream
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       201: IsoInfoResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       403: ErrorResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       415: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.POST("/isos/:name",
		func(c *gin.Context) {
			uploadIso(c, f.FileRoot, c.Param(`name`), f.dt)
		})
	// swagger:route DELETE /isos/{path} Isos deleteIso
	//
	// Delete an iso to a specific {path} in the tree under isos.
	//
	// The iso will be removed from the {path} in /isos.
	//
	//     Responses:
	//       204: NoContentResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/isos/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			isoName := path.Join(f.FileRoot, `isos`, path.Base(name))
			if err := os.Remove(isoName); err != nil {
				c.JSON(http.StatusNotFound,
					backend.NewError("API ERROR", http.StatusNotFound, fmt.Sprintf("delete: unable to delete %s: %v", name, err)))
				return
			}
			c.JSON(http.StatusAccepted, nil)
		})
}

func reloadBootenvsForIso(dt DTI, name string) {
	for _, blob := range dt.FetchAll(dt.NewBootEnv()) {
		env := backend.AsBootEnv(blob)
		if env.Available || env.OS.IsoFile != name {
			continue
		}
		env.Available = true
		dt.Update(env)
	}
}

func uploadIso(c *gin.Context, fileRoot, name string, dt DTI) {
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
