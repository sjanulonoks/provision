package backend

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
)

// RenderData is the struct that is passed to templates as a source of
// parameters and useful methods.
type RenderData struct {
	Machine *Machine // The Machine that the template is being rendered for.
	Env     *BootEnv // The boot environment that provided the template.
	p       *DataTracker
}

func (r *RenderData) DataTrackerAddress() string {
	return r.p.Address.String()
}

func (r *RenderData) DataTrackerURL() string {
	return r.p.FileURL
}

func (r *RenderData) CommandURL() string {
	return r.p.CommandURL
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

// Param is a helper function for extracting a parameter from Machine.Params
func (r *RenderData) Param(key string) (interface{}, error) {
	res, ok := r.Machine.Params[key]
	if !ok {
		return nil, fmt.Errorf("No such machine parameter %s", key)
	}
	return res, nil
}

func (r *RenderData) render(e *Error) []RenderedTemplate {
	var missingParams []string
	for _, param := range r.Env.RequiredParams {
		if _, ok := r.Machine.Params[param]; !ok {
			missingParams = append(missingParams, param)
		}
	}
	if len(missingParams) > 0 {
		e.Errorf("missing required machine params for %s:\n %v", r.Machine.Name, missingParams)
		return nil
	}
	res := make([]RenderedTemplate, len(r.Env.Templates))

	for i := range r.Env.Templates {
		ti := &r.Env.Templates[i]
		rt := RenderedTemplate{}
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
		rt.Template = ti.contents
		res[i] = rt
	}
	return res
}
