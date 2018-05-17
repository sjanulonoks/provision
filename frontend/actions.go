package frontend

import (
	"net/http"

	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/midlayer"
	"github.com/digitalrebar/provision/models"
	"github.com/gin-gonic/gin"
)

// ActionResponse return on a successful GET of a single Action
// swagger:response
type ActionResponse struct {
	// in: body
	Body *models.AvailableAction
}

// ActionsResponse return on a successful GET of all Actions
// swagger:response
type ActionsResponse struct {
	// in: body
	Body []*models.AvailableAction
}

// ActionPostResponse return on a successful POST of action
// swagger:response
type ActionPostResponse struct {
	// in: body
	Body interface{}
}

func (f *Frontend) makeActionEndpoints(cmdSet string, obj models.Model, idKey string) (
	getActions, getAction, runAction func(c *gin.Context)) {
	plugin := func(c *gin.Context) string {
		return c.Query("plugin")
	}
	idrtkeyok := func(c *gin.Context, op string) (string, *backend.RequestTracker, string, bool) {
		if op == "" {
			op = "action:" + c.Param("cmd")
		}
		id := c.Param(idKey)
		if id == "" {
			id = "global"
		}
		return id,
			f.rt(c, obj.(Lockable).Locks("actions")...),
			c.Param("cmd"),
			f.assureSimpleAuth(c, cmdSet, op, id)
	}

	return /* allActions */ func(c *gin.Context) {
			id, rt, _, ok := idrtkeyok(c, "actions")
			if !ok {
				return
			}
			actions := []models.AvailableAction{}
			ref := f.Find(c, rt, obj.Prefix(), id)
			if ref == nil {
				return
			}
			p := plugin(c)
			for _, laa := range f.pc.Actions.List(cmdSet) {
				for _, aa := range laa {
					if p != "" && p != aa.Plugin.Plugin.Name {
						continue
					}
					ma := &models.Action{
						Model:   ref,
						Command: aa.Command,
						Plugin:  aa.Plugin.Plugin.Name,
						Params:  map[string]interface{}{},
					}
					if _, err := validateAction(f, rt, cmdSet, id, ma); err == nil {
						actions = append(actions, aa.AvailableAction)
						break
					}
				}
			}
			c.JSON(http.StatusOK, actions)
		},
		/* oneAction */ func(c *gin.Context) {
			id, rt, cmd, ok := idrtkeyok(c, "")
			if !ok {
				return
			}
			ref := f.Find(c, rt, obj.Prefix(), id)
			if ref == nil {
				return
			}
			err := &models.Error{
				Code:  http.StatusNotFound,
				Model: obj.Prefix(),
				Key:   id,
				Type:  c.Request.Method,
			}
			err.Errorf("%s: Not Found", cmd)
			p := plugin(c)
			laa, _ := f.pc.Actions.Get(cmdSet, cmd)
			for _, aa := range laa {
				if p != "" && p != aa.Plugin.Plugin.Name {
					continue
				}
				ma := &models.Action{
					Model:   ref,
					Command: aa.Command,
					Plugin:  aa.Plugin.Plugin.Name,
					Params:  map[string]interface{}{},
				}
				if _, err = validateAction(f, rt, cmdSet, id, ma); err == nil {
					c.JSON(http.StatusOK, aa.AvailableAction)
					return
				}
			}
			c.AbortWithStatusJSON(err.Code, err)
		},
		/* runAction */ func(c *gin.Context) {
			var val map[string]interface{}
			if !assureDecode(c, &val) {
				return
			}
			id, rt, cmd, ok := idrtkeyok(c, "")
			if !ok {
				return
			}
			ref := f.Find(c, rt, obj.Prefix(), id)
			if ref == nil {
				return
			}
			res := &models.Action{
				Model:   ref,
				Plugin:  plugin(c),
				Command: cmd,
				Params:  val}
			ma, err := validateAction(f, rt, cmdSet, id, res)
			if err != nil {
				err.Type = "INVOKE"
				c.JSON(err.Code, err)
				return
			}
			rt.Publish(cmdSet, cmd, id, ma)
			retval, runErr := f.pc.Actions.Run(rt, cmdSet, ma)
			if runErr != nil {
				be, ok := runErr.(*models.Error)
				if !ok {
					c.JSON(409, runErr)
				} else {
					c.JSON(be.Code, be)
				}
			} else {
				c.JSON(http.StatusOK, retval)
			}
		}
}

func validateActionParameters(f *Frontend,
	rt *backend.RequestTracker,
	ma *models.Action,
	aa *midlayer.AvailableAction,
	err *models.Error) {

	name := ma.Command
	val := ma.Params

	m, _ := ma.Model.(models.Paramer)

	for _, param := range aa.RequiredParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			if m != nil {
				obj, ok = rt.GetParam(m, param, true)
			}
			if !ok {
				if o := rt.Find("profiles", f.dt.GlobalProfileName); o != nil {
					p := backend.AsProfile(o)
					if tobj, ok := p.Params[param]; ok {
						obj = tobj
					}
				}
			}

			// GREG: Default?

			// Put into place
			if obj != nil {
				val[param] = obj
			}
		}
		if obj == nil {
			err.Errorf("Action %s Missing Parameter %s", name, param)
		} else {
			pobj := rt.Find("params", param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj, nil); ev != nil {
					err.Errorf("Action %s: Invalid Parameter: %s: %s", name, param, ev.Error())
				}
			}
		}
	}
	for _, param := range aa.OptionalParams {
		var obj interface{} = nil
		obj, ok := val[param]
		if !ok {
			if m != nil {
				obj, ok = rt.GetParam(m, param, true)
			}
			if !ok {
				if o := rt.Find("profiles", f.dt.GlobalProfileName); o != nil {
					p := backend.AsProfile(o)
					if tobj, ok := p.Params[param]; ok {
						obj = tobj
					}
				}
			}

			// Put into place
			if obj != nil {
				val[param] = obj
			}
		}
		if obj != nil {
			pobj := rt.Find("params", param)
			if pobj != nil {
				rp := backend.AsParam(pobj)

				if ev := rp.ValidateValue(obj, nil); ev != nil {
					err.Errorf("Action %s: Invalid Parameter: %s: %s", name, param, ev.Error())
				}
			}
		}
	}
}

func validateAction(f *Frontend,
	rt *backend.RequestTracker,
	ob string,
	id string,
	ma *models.Action) (*models.Action, *models.Error) {

	cmd := ma.Command
	err := &models.Error{
		Code:  http.StatusBadRequest,
		Type:  "GET",
		Model: ob,
		Key:   id,
	}

	lraa := midlayer.AvailableActions{}
	var ok bool
	if ma.Plugin != "" {
		if aa, ok := f.pc.Actions.GetSpecific(ob, cmd, ma.Plugin); !ok {
			err.Errorf("Action %s on %s for plugin %s not found", cmd, ob, ma.Plugin)
			return nil, err
		} else {
			lraa = append(lraa, aa)
		}
	} else {
		if lraa, ok = f.pc.Actions.Get(ob, cmd); !ok {
			err.Errorf("Action %s on %s: Not Found", cmd, ob)
			return nil, err
		}
	}

	for _, aa := range lraa {
		err = &models.Error{
			Code:  http.StatusBadRequest,
			Type:  "GET",
			Model: ob,
			Key:   id,
		}
		rt.Do(func(_ backend.Stores) {
			validateActionParameters(f, rt, ma, aa, err)
		})

		if !err.ContainsError() {
			ma.Plugin = aa.Plugin.Plugin.Name
			return ma, nil
		}
	}
	return nil, err
}
