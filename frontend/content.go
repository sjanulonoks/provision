package frontend

import (
	"fmt"
	"net/http"

	"github.com/digitalrebar/provision/midlayer"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	"github.com/gin-gonic/gin"
)

// ContentsResponse returned on a successful GET of a contents
// swagger:response
type ContentResponse struct {
	// in: body
	Body *models.Content
}

// ContentSummaryResponse returned on a successful Post of a content
// swagger:response
type ContentSummaryResponse struct {
	// in: body
	Body *models.ContentSummary
}

// ContentsResponse returned on a successful GET of all contents
// swagger:response
type ContentsResponse struct {
	// in: body
	Body []*models.ContentSummary
}

// swagger:parameters uploadContent createContent
type ContentBodyParameter struct {
	// in: body
	Body *models.Content
}

// swagger:parameters getContent deleteContent uploadContent
type ContentParameter struct {
	// in: path
	Name string `json:"name"`
}

func (f *Frontend) buildNewStore(content *models.Content) (newStore store.Store, err error) {
	filename := fmt.Sprintf("file:///%s/%s-%s.yaml?codec=yaml", f.SaasDir, content.Meta.Name, content.Meta.Version)

	newStore, err = store.Open(filename)
	if err != nil {
		return
	}

	if md, ok := newStore.(store.MetaSaver); ok {
		data := map[string]string{
			"Name":        content.Meta.Name,
			"Source":      content.Meta.Source,
			"Description": content.Meta.Description,
			"Version":     content.Meta.Version,
			"Type":        "dynamic",
		}
		md.SetMetaData(data)
	}

	for prefix, objs := range content.Sections {
		var sub store.Store
		sub, err = newStore.MakeSub(prefix)
		if err != nil {
			return
		}

		for k, obj := range objs {
			err = sub.Save(k, obj)
			if err != nil {
				return
			}
		}
	}

	return
}

func buildSummary(st store.Store) *models.ContentSummary {
	mst, ok := st.(store.MetaSaver)
	if !ok {
		return nil
	}
	cs := &models.ContentSummary{}
	cs.Fill()
	metaData := mst.MetaData()

	cs.Meta.Name = metaData["Name"]
	cs.Meta.Source = metaData["Source"]
	cs.Meta.Description = metaData["Description"]
	cs.Meta.Version = metaData["Version"]
	if val, ok := metaData["Type"]; ok {
		cs.Meta.Type = val
	} else {
		cs.Meta.Type = "dynamic"
	}
	cs.Meta.Writable = false
	cs.Meta.Overwritable = false
	if cs.Meta.Name == "BackingStore" {
		cs.Meta.Type = "writable"
		cs.Meta.Writable = true
	} else if cs.Meta.Name == "LocalStore" {
		cs.Meta.Type = "local"
	} else if cs.Meta.Name == "BasicStore" {
		cs.Meta.Type = "basic"
	} else if cs.Meta.Name == "DefaultStore" {
		cs.Meta.Type = "default"
		cs.Meta.Overwritable = true
	}
	subs := mst.Subs()
	for k, sub := range subs {
		keys, err := sub.Keys()
		if err != nil {
			continue
		}
		cs.Counts[k] = len(keys)
	}

	return cs
}

func (f *Frontend) buildContent(st store.Store) (*models.Content, *models.Error) {
	content := &models.Content{}

	var md map[string]string
	mst, ok := st.(store.MetaSaver)
	if ok {
		md = mst.MetaData()
	} else {
		md = map[string]string{}
	}

	// Copy in MetaData
	if val, ok := md["Name"]; ok {
		content.Meta.Name = val
	} else {
		content.Meta.Name = "Unknown"
	}
	if val, ok := md["Source"]; ok {
		content.Meta.Source = val
	} else {
		content.Meta.Source = "Unknown"
	}
	if val, ok := md["Description"]; ok {
		content.Meta.Description = val
	} else {
		content.Meta.Description = "Unknown"
	}
	if val, ok := md["Version"]; ok {
		content.Meta.Version = val
	} else {
		content.Meta.Version = "Unknown"
	}
	if val, ok := md["Type"]; ok {
		content.Meta.Type = val
	} else {
		content.Meta.Type = "dynamic"
	}

	content.Meta.Writable = false
	content.Meta.Overwritable = false
	if content.Meta.Name == "BackingStore" {
		content.Meta.Type = "writable"
		content.Meta.Writable = true
	} else if content.Meta.Name == "LocalStore" {
		content.Meta.Type = "local"
	} else if content.Meta.Name == "BasicStore" {
		content.Meta.Type = "basic"
	} else if content.Meta.Name == "DefaultStore" {
		content.Meta.Type = "default"
		content.Meta.Overwritable = true
	}

	// Walk subs to build content sets
	content.Sections = models.Sections{}
	for prefix, sub := range st.Subs() {
		_, err := models.New(prefix)
		if err != nil {
			berr := models.NewError("ValidationError", http.StatusUnprocessableEntity, err.Error())
			return nil, berr
		}

		keys, err := sub.Keys()
		if err != nil {
			berr := models.NewError("ServerError", http.StatusInternalServerError, err.Error())
			return nil, berr
		}
		objs := make(models.Section, 0)
		for _, k := range keys {
			// This is protected by the earlier check
			v, _ := models.New(prefix)
			err := sub.Load(k, &v)
			if err != nil {
				berr := models.NewError("ServerError", http.StatusInternalServerError, err.Error())
				return nil, berr
			}
			objs[k] = v
		}

		content.Sections[prefix] = objs
	}

	return content, nil
}

func (f *Frontend) findContent(name string) (cst store.Store) {
	if stack, ok := f.dt.Backend.(*midlayer.DataStack); !ok {
		mst, ok := f.dt.Backend.(store.MetaSaver)
		if !ok {
			return nil
		}
		metaData := mst.MetaData()
		if metaData["Name"] == name {
			cst = f.dt.Backend
		}
	} else {
		for _, st := range stack.Layers() {
			mst, ok := st.(store.MetaSaver)
			if !ok {
				continue
			}
			metaData := mst.MetaData()
			if metaData["Name"] == name {
				cst = st
				break
			}
		}
	}
	return
}

func (f *Frontend) InitContentApi() {
	// swagger:route GET /contents Contents listContents
	//
	// Lists possible contents on the system to serve DHCP
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       200: ContentsResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/contents",
		func(c *gin.Context) {
			if !f.assureAuth(c, "contents", "list", "") {
				return
			}

			contents := []*models.ContentSummary{}
			func() {
				_, unlocker := f.dt.LockAll()
				defer unlocker()

				if stack, ok := f.dt.Backend.(*midlayer.DataStack); !ok {
					cs := buildSummary(f.dt.Backend)
					if cs != nil {
						contents = append(contents, cs)
					}
				} else {
					for _, st := range stack.Layers() {
						cs := buildSummary(st)
						if cs != nil {
							contents = append(contents, cs)
						}
					}
				}
			}()

			c.JSON(http.StatusOK, contents)
		})

	// swagger:route GET /contents/{name} Contents getContent
	//
	// Get a specific content with {name}
	//
	// Get a specific content specified by {name}.
	//
	//     Produces:
	//       application/json
	//
	//     Responses:
	//       200: ContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       500: ErrorResponse
	f.ApiGroup.GET("/contents/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			if !f.assureAuth(c, "contents", "get", name) {
				return
			}

			func() {
				_, unlocker := f.dt.LockAll()
				defer unlocker()

				if cst := f.findContent(name); cst == nil {
					res := &models.Error{
						Model: "contents",
						Key:   name,
						Type:  c.Request.Method,
						Code:  http.StatusNotFound,
					}
					res.Errorf("No such content store")
					c.JSON(http.StatusNotFound, res)
				} else {
					content, err := f.buildContent(cst)
					if err != nil {
						c.JSON(err.Code, err)
					} else {
						c.JSON(http.StatusOK, content)
					}
				}
			}()
		})

	// swagger:route POST /contents Contents createContent
	//
	// Create content into Digital Rebar Provision
	//
	//     Responses:
	//       201: ContentSummaryResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       403: ErrorResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       415: ErrorResponse
	//       422: ErrorResponse
	//       500: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.POST("/contents",
		func(c *gin.Context) {
			if !f.assureAuth(c, "contents", "create", "*") {
				return
			}
			content := &models.Content{}
			if !assureDecode(c, content) {
				return
			}

			name := content.Meta.Name
			func() {
				_, unlocker := f.dt.LockAll()
				defer unlocker()

				if cst := f.findContent(name); cst != nil {
					res := &models.Error{
						Model: "contents",
						Key:   name,
						Type:  c.Request.Method,
						Code:  http.StatusConflict,
					}
					res.Errorf("Content %s already exists", name)
					c.JSON(http.StatusConflict, res)
					return
				}

				if newStore, err := f.buildNewStore(content); err != nil {
					jsonError(c, err, http.StatusInternalServerError,
						fmt.Sprintf("failed to build content: %s: ", name))
					return
				} else {
					cs := buildSummary(newStore)

					ds := f.dt.Backend.(*midlayer.DataStack)
					if nbs, hard, soft := ds.AddReplaceSAAS(name, newStore, f.Logger, nil); hard != nil {
						midlayer.CleanUpStore(newStore)
						jsonError(c, hard, http.StatusInternalServerError,
							fmt.Sprintf("failed to add content: %s: ", name))
					} else {
						if soft != nil {
							if berr, ok := soft.(*models.Error); ok {
								cs.Warnings = berr.Messages
							}
						}
						f.dt.ReplaceBackend(nbs)
						c.JSON(http.StatusCreated, cs)
					}
				}
			}()
		})

	// swagger:route PUT /contents/{name} Contents uploadContent
	//
	// Replace content in Digital Rebar Provision
	//
	//     Responses:
	//       200: ContentSummaryResponse
	//       400: ErrorResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       403: ErrorResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       415: ErrorResponse
	//       422: ErrorResponse
	//       500: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.PUT("/contents/:name",
		func(c *gin.Context) {
			if !f.assureAuth(c, "contents", "update", "*") {
				return
			}
			content := &models.Content{}
			if !assureDecode(c, content) {
				return
			}

			name := c.Param(`name`)
			if name != content.Meta.Name {
				res := &models.Error{
					Model: "contents",
					Key:   name,
					Type:  c.Request.Method,
					Code:  http.StatusBadRequest,
				}
				res.Errorf("Cannot change name from %s to %s", name, content.Meta.Name)
				c.JSON(http.StatusBadRequest, res)
				return

			}

			func() {
				_, unlocker := f.dt.LockAll()
				defer unlocker()

				if cst := f.findContent(name); cst == nil {
					res := &models.Error{
						Model: "contents",
						Key:   name,
						Type:  c.Request.Method,
						Code:  http.StatusNotFound,
					}
					res.Errorf("Cannot find %s", name)
					c.JSON(http.StatusNotFound, res)
					return
				}

				if newStore, err := f.buildNewStore(content); err != nil {
					res := &models.Error{
						Model: "contents",
						Key:   name,
						Type:  c.Request.Method,
						Code:  http.StatusInternalServerError,
					}
					res.Errorf("Failed to build content")
					res.AddError(err)
					c.JSON(res.Code, res)
					return
				} else {
					cs := buildSummary(newStore)

					ds := f.dt.Backend.(*midlayer.DataStack)
					if nbs, hard, soft := ds.AddReplaceSAAS(name, newStore, f.Logger, nil); hard != nil {
						midlayer.CleanUpStore(newStore)
						res := &models.Error{
							Model: "contents",
							Key:   name,
							Type:  c.Request.Method,
							Code:  http.StatusInternalServerError,
						}
						res.AddError(hard)
						res.AddError(soft)
						c.JSON(res.Code, res)
						return
					} else {
						if soft != nil {
							if berr, ok := soft.(*models.Error); ok {
								cs.Warnings = berr.Messages
							}
						}
						f.dt.ReplaceBackend(nbs)
						c.JSON(http.StatusOK, cs)
					}
				}
			}()
		})

	// swagger:route DELETE /contents/{name} Contents deleteContent
	//
	// Delete a content set.
	//
	//     Responses:
	//       204: NoContentResponse
	//       401: NoContentResponse
	//       403: NoContentResponse
	//       404: ErrorResponse
	//       409: ErrorResponse
	//       422: ErrorResponse
	f.ApiGroup.DELETE("/contents/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			if !f.assureAuth(c, "contents", "delete", name) {
				return
			}

			func() {
				_, unlocker := f.dt.LockAll()
				defer unlocker()

				cst := f.findContent(name)
				if cst == nil {
					res := &models.Error{
						Model:    "contents",
						Key:      name,
						Type:     c.Request.Method,
						Messages: []string{"No such content store"},
						Code:     http.StatusNotFound,
					}
					c.JSON(http.StatusNotFound, res)
					return
				}

				ds := f.dt.Backend.(*midlayer.DataStack)
				if nbs, hard, _ := ds.RemoveSAAS(name, f.Logger); hard != nil {
					jsonError(c, hard, http.StatusInternalServerError,
						fmt.Sprintf("failed to remove content: %s: ", name))
				} else {
					f.dt.ReplaceBackend(nbs)
					c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
				}
			}()
		})
}
