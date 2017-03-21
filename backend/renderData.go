package backend

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// RenderTemplate is the result of rendering a BootEnv template
type renderedTemplate struct {
	// Path is the absolute path that the Template will be rendered to.
	Path string
	// Template is the template that will rendered
	Template *Template
	// Vars holds the variables that will be used during template expansion.
	Vars *RenderData
}

func (r *renderedTemplate) mkdirAll() error {
	return os.MkdirAll(path.Dir(r.Path), 0755)
}

func (r *renderedTemplate) write(e *Error) {
	tmplDest, err := os.Create(r.Path)
	if err != nil {
		e.Errorf("Unable to create file %s: %v", r.Path, err)
		return
	}
	defer tmplDest.Close()
	if err := r.Template.render(tmplDest, r.Vars); err != nil {
		os.Remove(r.Path)
		e.Errorf("Error rendering template %s: %v", r.Template.Key(), err)
		return
	}
	tmplDest.Sync()
}

func (r *renderedTemplate) remove(e *Error) {
	if r.Path != "" {
		if err := os.Remove(r.Path); err != nil {
			e.Errorf("%v", err)
		}
	}
}

// RenderData is the struct that is passed to templates as a source of
// parameters and useful methods.
type RenderData struct {
	Machine           *Machine // The Machine that the template is being rendered for.
	Env               *BootEnv // The boot environment that provided the template.
	renderedTemplates []renderedTemplate
	p                 *DataTracker
}

func (r *RenderData) ProvisionerAddress() string {
	return r.p.OurAddress
}

func (r *RenderData) ProvisionerURL() string {
	return r.p.FileURL
}

func (r *RenderData) CommandURL() string {
	return r.p.CommandURL
}

func (r *RenderData) ApiURL() string {
	return r.p.ApiURL
}

// BootParams is a helper function that expands the BootParams
// template from the boot environment.
func (r *RenderData) BootParams() (string, error) {
	res := &bytes.Buffer{}
	if r.Env.bootParamsTmpl == nil {
		return "", nil
	}
	if err := r.Env.bootParamsTmpl.Execute(res, r); err != nil {
		return "", err
	}
	return res.String(), nil
}

func (r *RenderData) ParseUrl(segment, rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	switch segment {
	case "scheme":
		return parsedUrl.Scheme, nil
	case "host":
		return parsedUrl.Host, nil
	case "path":
		return parsedUrl.Path, nil
	}
	return "", fmt.Errorf("No idea how to get URL part %s from %s", segment, rawUrl)
}

// ParamExists is a helper function for determining the existence of a machine parameter.
func (r *RenderData) ParamExists(key string) bool {
	_, ok := r.Machine.Params[key]
	return ok
}

// Param is a helper function for extracting a parameter from Machine.Params
func (r *RenderData) Param(key string) (interface{}, error) {
	res, ok := r.Machine.Params[key]
	if ok {
		return res, nil
	}
	param := r.p.load("parameters", key)
	if param != nil {
		return AsParam(param).Value, nil
	}
	return nil, fmt.Errorf("No such machine parameter %s", key)
}

func (r *RenderData) render(e *Error) {
	var missingParams []string
	if len(r.Env.RequiredParams) > 0 && (r.Machine == nil || r.Machine.Params == nil) {
		e.Errorf("Machine is nil or does not have params")
		return
	}
	for _, param := range r.Env.RequiredParams {
		if _, ok := r.Machine.Params[param]; !ok {
			globalParam := r.p.load("parameters", param)
			if globalParam == nil {
				missingParams = append(missingParams, param)
			}
		}
	}
	if len(missingParams) > 0 {
		e.Errorf("missing required machine params for %s:\n %v", r.Machine.Name, missingParams)
		return
	}
	r.renderedTemplates = make([]renderedTemplate, len(r.Env.Templates))

	for i := range r.Env.Templates {
		ti := &r.Env.Templates[i]
		rt := renderedTemplate{}
		tmpl, found := ti.contents(r.p)
		if !found {
			e.Errorf("Template does not exist: %s", ti.ID)
			continue
		}
		// first, render the path
		buf := &bytes.Buffer{}
		if err := ti.pathTmpl.Execute(buf, r); err != nil {
			e.Errorf("Error rendering template %s path %s: %v",
				ti.Name,
				ti.Path,
				err)
		} else {
			rt.Path = filepath.Join(r.p.FileRoot, buf.String())
		}
		rt.Template = tmpl
		rt.Vars = r
		r.renderedTemplates[i] = rt
	}
}

func (r *RenderData) mkPaths(e *Error) {
	for _, rt := range r.renderedTemplates {
		if rt.Path != "" {
			if err := rt.mkdirAll(); err != nil {
				e.Errorf("%v", err)
			}
		}
	}
}

func (r *RenderData) remove(e *Error) {
	for _, rt := range r.renderedTemplates {
		rt.remove(e)
	}
}

func (r *RenderData) write(e *Error) {
	for _, rt := range r.renderedTemplates {
		rt.write(e)
	}
}
