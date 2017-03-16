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

type FilePaths []string

// FilesResponse returned on a successful GET of files
// swagger:response
type FilesResponse struct {
	// in: body
	Body FilePaths
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

// FileResponse returned on a successful GET of a file
// swagger:response
type FileResponse struct {
	// in: body
	Body interface{}
}

type FileInfo struct {
	Path string
	Size int64
}

// FileInfoResponse returned on a successful upload of a file
// swagger:response
type FileInfoResponse struct {
	// in: body
	Body *FileInfo
}

// swagger:parameters listFiles
type FilesPathQueryParameter struct {
	// in: query
	Path string `json:"path"`
}

// swagger:parameters uploadFile getFile deleteFile
type FilePathPathParameter struct {
	// in: path
	Path string `json:"path"`
}

// FileData body of the upload
// swagger:parameters uploadFile
type FileData struct {
	// in: body
	Body interface{}
}

func (f *Frontend) InitFileApi() {
	// swagger:route GET /files Files listFiles
	//
	// Lists files in files directory or subdirectory per query parameter
	//
	// Lists the files in a directory under /files.  path=<path to return>
	// Path defaults to /
	//
	//     Responses:
	//       200: FilesResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/files",
		func(c *gin.Context) {
			pathPart, _ := c.GetQuery("path")
			ents, err := ioutil.ReadDir(path.Join(f.FileRoot, "files", path.Clean(pathPart)))
			if err != nil {
				c.JSON(http.StatusNotFound,
					backend.NewError("API ERROR", http.StatusNotFound,
						fmt.Sprintf("list: error listing files: %v", err)))
				return
			}
			res := make(FilePaths, 0, 0)
			for _, ent := range ents {
				if ent.Mode().IsRegular() {
					res = append(res, ent.Name())
				} else if ent.Mode().IsDir() {
					res = append(res, ent.Name()+"/")
				}
			}
			c.JSON(http.StatusOK, res)
		})

	// swagger:route GET /files/{path} Files getFile
	//
	// Get a specific File with {path}
	//
	// Get a specific file specified by {path} under files.
	//
	//     Produces:
	//       application/octet-stream
	//       application/json
	//
	//     Responses:
	//       200: FileResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.GET("/files/*path",
		func(c *gin.Context) {
			fileName := path.Join(f.FileRoot, `files`, path.Clean(c.Param(`path`)))
			c.Writer.Header().Set("Content-Type", "application/octet-stream")
			c.File(fileName)
		})

	// swagger:route POST /files/{path} Files uploadFile
	//
	// Upload a file to a specific {path} in the tree under files.
	//
	// The file will be uploaded to the {path} in /files.  The {path} will be created.
	//
	//     Consumes:
	//       application/octet-stream
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       201: FileInfoResponse
	//       400: ErrorResponse
	//       401: ErrorResponse
	//       403: ErrorResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       415: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.POST("/files/*path",
		func(c *gin.Context) {
			name := c.Param(`path`)
			if c.Request.Header.Get(`Content-Type`) != `application/octet-stream` {
				c.JSON(http.StatusUnsupportedMediaType,
					backend.NewError("API ERROR", http.StatusUnsupportedMediaType,
						fmt.Sprintf("upload: file %s must have content-type application/octet-stream", name)))
				return
			}
			fileTmpName := path.Join(f.FileRoot, `files`, fmt.Sprintf(`.%s.part`, path.Clean(name)))
			fileName := path.Join(f.FileRoot, `files`, path.Clean(name))
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

			if c.Request.ContentLength > 0 && copied != c.Request.ContentLength {
				os.Remove(fileTmpName)
				c.JSON(http.StatusBadRequest,
					backend.NewError("API ERROR", http.StatusBadRequest,
						fmt.Sprintf("upload: Failed to upload entire file %s: %d bytes expected, %d bytes recieved", name, c.Request.ContentLength, copied)))
				return
			}
			os.Remove(fileName)
			os.Rename(fileTmpName, fileName)
			c.JSON(http.StatusCreated, &FileInfo{Path: name, Size: copied})
		})

	// swagger:route DELETE /files/{path} Files deleteFile
	//
	// Delete a file to a specific {path} in the tree under files.
	//
	// The file will be removed from the {path} in /files.
	//
	//     Responses:
	//       204: NoContentResponse
	//       401: ErrorResponse
	//       404: ErrorResponse
	f.ApiGroup.DELETE("/files/*path",
		func(c *gin.Context) {
			fileName := path.Join(f.FileRoot, `files`, path.Clean(c.Param(`path`)))
			if fileName == path.Join(f.FileRoot, `files`) {
				c.JSON(http.StatusForbidden,
					backend.NewError("API ERROR", http.StatusForbidden,
						fmt.Sprintf("delete: Not allowed to remove files dir")))
				return
			}
			if err := os.Remove(fileName); err != nil {
				c.JSON(http.StatusNotFound,
					backend.NewError("API ERROR", http.StatusNotFound,
						fmt.Sprintf("delete: unable to delete %s: %v", c.Param(`path`), err)))
				return
			}
			c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
		})

}
