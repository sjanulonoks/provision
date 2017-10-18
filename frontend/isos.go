package frontend

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
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

// IsoInfoResponse returned on a successful upload of an iso
// swagger:response
type IsoInfoResponse struct {
	// in: body
	Body *models.BlobInfo
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/isos",
		func(c *gin.Context) {
			if !f.assureAuth(c, "isos", "list", "") {
				return
			}
			ents, err := ioutil.ReadDir(path.Join(f.FileRoot, "isos"))
			if err != nil {
				c.JSON(http.StatusNotFound,
					models.NewError("API ERROR", http.StatusNotFound, fmt.Sprintf("list: error listing isos: %v", err)))
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/isos/:name",
		func(c *gin.Context) {
			if !f.assureAuth(c, "isos", "get", c.Param(`name`)) {
				return
			}
			fileName := path.Join(f.FileRoot, `isos`, path.Clean(c.Param(`name`)))
			if st, err := os.Stat(fileName); err != nil || !st.Mode().IsRegular() {
				res := &models.Error{
					Code:  http.StatusNotFound,
					Key:   c.Param(`name`),
					Model: "isos",
					Type:  c.Request.Method,
				}
				res.Errorf("Not a regular file")
				c.JSON(res.Code, res)
				return
			}
			c.File(fileName)
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       415: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.POST("/isos/:name",
		func(c *gin.Context) {
			if !f.assureAuth(c, "isos", "post", c.Param(`name`)) {
				return
			}
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
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/isos/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			if !f.assureAuth(c, "isos", "delete", name) {
				return
			}
			isoName := path.Join(f.FileRoot, `isos`, path.Base(name))
			if err := os.Remove(isoName); err != nil {
				res := &models.Error{
					Code:  http.StatusNotFound,
					Model: "isos",
					Key:   name,
				}
				res.Errorf("no such iso")
				c.JSON(res.Code, res)
				return
			}
			c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
		})
}

func reloadBootenvsForIso(dt *backend.DataTracker, name string) {
	ref := &backend.BootEnv{}
	d, unloader := dt.LockEnts(ref.Locks("update")...)
	defer unloader()

	for _, blob := range d("bootenvs").Items() {
		env := backend.AsBootEnv(blob)
		if env.Available || env.OS.IsoFile != name {
			continue
		}
		env.Available = true
		dt.Update(d, env)
	}
}

func uploadIso(c *gin.Context, fileRoot, name string, dt *backend.DataTracker) {
	if err := os.MkdirAll(path.Join(fileRoot, `isos`), 0755); err != nil {
		c.JSON(http.StatusConflict,
			models.NewError(c.Request.Method, http.StatusConflict, fmt.Sprintf("upload: unable to create isos directory")))
		return
	}
	var copied int64

	ctype := c.Request.Header.Get(`Content-Type`)
	switch strings.Split(ctype, "; ")[0] {
	case `application/octet-stream`:
		if c.Request.Body == nil {
			c.JSON(http.StatusBadRequest,
				models.NewError("API ERROR", http.StatusBadRequest,
					fmt.Sprintf("upload: Unable to upload %s: missing body", name)))
			return
		}
	case `multipart/form-data`:
		header, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest,
				models.NewError("API ERROR", http.StatusBadRequest,
					fmt.Sprintf("upload: Failed to find multipart file: %v", err)))
			return
		}
		name = path.Base(header.Filename)
	default:
		c.JSON(http.StatusUnsupportedMediaType,
			models.NewError("API ERROR", http.StatusUnsupportedMediaType,
				fmt.Sprintf("upload: iso %s content-type %s is not application/octet-stream or multipart/form-data", name, ctype)))
		return
	}

	isoTmpName := path.Join(fileRoot, `isos`, fmt.Sprintf(`.%s.part`, path.Base(name)))
	isoName := path.Join(fileRoot, `isos`, path.Base(name))

	out, err := os.Create(isoTmpName)
	if err != nil {
		c.JSON(http.StatusConflict,
			models.NewError("API ERROR", http.StatusConflict, fmt.Sprintf("upload: iso %s already uploading", name)))
		return
	}
	defer out.Close()

	if err != nil {
		c.JSON(http.StatusConflict,
			models.NewError("API ERROR", http.StatusConflict, fmt.Sprintf("upload: Unable to upload %s: %v", name, err)))
		return
	}

	switch strings.Split(ctype, "; ")[0] {
	case `application/octet-stream`:
		copied, err = io.Copy(out, c.Request.Body)
		if c.Request.ContentLength > 0 && copied != c.Request.ContentLength {
			os.Remove(isoTmpName)
			c.JSON(http.StatusBadRequest,
				models.NewError("API ERROR", http.StatusBadRequest,
					fmt.Sprintf("upload: Failed to upload entire file %s: %d bytes expected, %d bytes received", name, c.Request.ContentLength, copied)))
			return
		}
		if err != nil {
			os.Remove(isoTmpName)
			c.JSON(http.StatusInsufficientStorage,
				models.NewError("API ERROR",
					http.StatusInsufficientStorage, fmt.Sprintf("upload: Failed to upload %s: %v", name, err)))
			return
		}
	case `multipart/form-data`:
		header, _ := c.FormFile("file")
		file, err := header.Open()
		defer file.Close()
		copied, err = io.Copy(out, file)
		if err != nil {
			c.JSON(http.StatusConflict,
				models.NewError("API ERROR", http.StatusBadRequest, fmt.Sprintf("upload: iso %s could not save", header.Filename)))
			return
		}
		file.Close()
	}

	os.Remove(isoName)
	os.Rename(isoTmpName, isoName)
	go reloadBootenvsForIso(dt, name)
	c.JSON(http.StatusCreated, &models.BlobInfo{Path: name, Size: copied})
}
