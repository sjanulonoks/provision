package frontend

import (
	"fmt"
	"net/http"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/store"
	"github.com/gin-gonic/gin"
)

//
// Isos???
// Contents??
//
// swagger:model
type Content struct {
	Name        string
	Description string
	Version     string

	/*
		        These are the sections:

			tasks        map[string]*backend.Task
			bootEnvs     map[string]*backend.BootEnv
			templates    map[string]*backend.Template
			profiles     map[string]*backend.Profile
			params       map[string]*backend.Param
			reservations map[string]*backend.Reservation
			subnets      map[string]*backend.Subnet
			users        map[string]*backend.User
			preferences  map[string]*backend.Pref
			plugins      map[string]*backend.Plugin
			machines     map[string]*backend.Machine
			leases       map[string]*backend.Lease
	*/
	Sections map[string]map[string]interface{}
}

// swagger:model
type ContentSummary struct {
	Name        string
	Description string
	Version     string
	Counts      map[string]int
}

// ContentsResponse returned on a successful GET of a contents
// swagger:response
type ContentResponse struct {
	// in: body
	Body *Content
}

// ContentSummaryResponse returned on a successful Post of a content
// swagger:response
type ContentSummaryResponse struct {
	// in: body
	Body *ContentSummary
}

// ContentsResponse returned on a successful GET of all contents
// swagger:response
type ContentsResponse struct {
	// in: body
	Body []*ContentSummary
}

// swagger:parameters getContent deleteContent
type ContentParameter struct {
	// in: path
	Name string `json:"name"`
}

func buildSummary(st store.Store) *ContentSummary {
	mst, ok := st.(store.MetaSaver)
	if !ok {
		return nil
	}
	cs := &ContentSummary{}
	metaData := mst.MetaData()

	cs.Name = metaData["Name"]
	cs.Description = metaData["Description"]
	cs.Version = metaData["Version"]
	cs.Counts = map[string]int{}

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

func (f *Frontend) buildContent(st store.Store) (*Content, *backend.Error) {
	// GREG: Locking
	content := &Content{}

	var md map[string]string
	mst, ok := st.(store.MetaSaver)
	if ok {
		md = mst.MetaData()
	} else {
		md = map[string]string{}
	}

	// Copy in MetaData
	if val, ok := md["Name"]; ok {
		content.Name = val
	} else {
		content.Name = "Unknown"
	}
	if val, ok := md["Description"]; ok {
		content.Description = val
	} else {
		content.Description = "Unknown"
	}
	if val, ok := md["Version"]; ok {
		content.Version = val
	} else {
		content.Version = "Unknown"
	}

	// Walk subs to build content sets
	content.Sections = map[string]map[string]interface{}{}
	for prefix, sub := range st.Subs() {
		obj := f.dt.NewKeySaver(prefix)

		keys, err := sub.Keys()
		if err != nil {
			berr := backend.NewError("ServerError", http.StatusInternalServerError, err.Error())
			return nil, berr
		}
		objs := make(map[string]interface{}, 0)
		for _, k := range keys {
			v := obj.New()
			// GREG: Should this go through the load hooks??
			err := sub.Load(k, &v)
			if err != nil {
				berr := backend.NewError("ServerError", http.StatusInternalServerError, err.Error())
				return nil, berr
			}
			objs[k] = v
		}

		content.Sections[prefix] = objs
	}

	return content, nil
}

func (f *Frontend) findContent(name string) (cst store.Store) {
	if stack, ok := f.dt.Backend.(*store.StackedStore); !ok {
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
			if !assureAuth(c, f.Logger, "contents", "list", "") {
				return
			}

			contents := []*ContentSummary{}

			if stack, ok := f.dt.Backend.(*store.StackedStore); !ok {
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
			if !assureAuth(c, f.Logger, "contents", "get", name) {
				return
			}

			if cst := f.findContent(name); cst == nil {
				c.JSON(http.StatusNotFound,
					backend.NewError("API_ERROR", http.StatusNotFound,
						fmt.Sprintf("content get: not found: %s", name)))
			} else {
				content, err := f.buildContent(cst)
				if err != nil {
					c.JSON(err.Code, err)
				} else {
					c.JSON(http.StatusOK, content)
				}
			}
		})

	// swagger:route POST /contents Contents uploadContent
	//
	// Upload content into Digital Rebar Provision (Create/Update)
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
	//       500: ErrorResponse
	//       507: ErrorResponse
	f.ApiGroup.POST("/contents",
		func(c *gin.Context) {
			if !assureAuth(c, f.Logger, "contents", "post", "*") {
				return
			}
			content := &Content{}
			if !assureDecode(c, content) {
				return
			}
			name := content.Name

			newStore, err := store.Open("file:///tmp/newstore")
			if err != nil {
				c.JSON(http.StatusInternalServerError,
					backend.NewError("API_ERROR", http.StatusInternalServerError,
						fmt.Sprintf("content load: error: %s: %v", name, err)))
				return
			}

			if md, ok := newStore.(store.MetaSaver); ok {
				data := map[string]string{
					"Name":        content.Name,
					"Description": content.Description,
					"Version":     content.Version,
				}
				md.SetMetaData(data)
			}

			for prefix, objs := range content.Sections {
				sub, err := newStore.MakeSub(prefix)
				if err != nil {
					c.JSON(http.StatusInternalServerError,
						backend.NewError("API_ERROR", http.StatusInternalServerError,
							fmt.Sprintf("content load: error: %s: %v", name, err)))
					return
				}

				for k, obj := range objs {
					err := sub.Save(k, obj)
					if err != nil {
						c.JSON(http.StatusInternalServerError,
							backend.NewError("API_ERROR", http.StatusInternalServerError,
								fmt.Sprintf("content load: error: %s: %v", name, err)))
						return
					}
				}
			}

			// GREG: Inject store

			cs := buildSummary(newStore)
			c.JSON(http.StatusCreated, cs)
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
	f.ApiGroup.DELETE("/contents/:name",
		func(c *gin.Context) {
			name := c.Param(`name`)
			if !assureAuth(c, f.Logger, "contents", "delete", name) {
				return
			}

			cst := f.findContent(name)
			if cst == nil {
				c.JSON(http.StatusNotFound,
					backend.NewError("API_ERROR", http.StatusNotFound,
						fmt.Sprintf("content get: not found: %s", name)))
				return
			}

			// GREG: Delete store layer.

			c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
		})
}
