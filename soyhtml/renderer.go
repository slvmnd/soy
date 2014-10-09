// Package soyhtml renders a compiled set of Soy to HTML.
package soyhtml

import (
	"errors"
	"io"

	"github.com/slvmnd/soy/ast"
	"github.com/slvmnd/soy/data"
	soyt "github.com/slvmnd/soy/template"
)

var ErrTemplateNotFound = errors.New("template not found")

// Renderer provides parameters to template execution.
// At minimum, Registry and Template are required to render a template..
type Renderer struct {
	tofu           *Tofu    // a registry of all templates in a bundle
	name           string   // fully-qualified name of the template to render
	variant        string   // optional variant argument for delegate template
	useDelTemplate bool     // set to render a delegate template
	ij             data.Map // data for the $ij map
}

// Inject sets the given data map as the $ij injected data.
func (r *Renderer) Inject(ij data.Map) *Renderer {
	r.ij = ij
	return r
}

// Execute applies a parsed template to the specified data object,
// and writes the output to wr.
func (t Renderer) Execute(wr io.Writer, obj data.Map) (err error) {
	if t.tofu == nil || t.tofu.registry == nil {
		return errors.New("Template Registry required")
	}
	if t.name == "" {
		return errors.New("Template name required")
	}

	var tmpl soyt.Template
	var ok bool

	if t.useDelTemplate {
		var calledTmpl, ok = t.tofu.registry.DelTemplate(t.name, t.variant)
		if !ok {
			return ErrTemplateNotFound
		}

		tmpl = soyt.Template{
			Doc:       calledTmpl.Doc,
			Node:      &calledTmpl.Node.TemplateNode,
			Namespace: calledTmpl.Namespace,
		}
	} else {
		tmpl, ok = t.tofu.registry.Template(t.name)
		if !ok {
			return ErrTemplateNotFound
		}
	}

	var autoescapeMode = tmpl.Namespace.Autoescape
	if autoescapeMode == ast.AutoescapeUnspecified {
		autoescapeMode = ast.AutoescapeOn
	}

	var initialScope = newScope(obj)
	initialScope.enter()

	state := &state{
		tmpl:       tmpl,
		registry:   *t.tofu.registry,
		namespace:  tmpl.Namespace.Name,
		autoescape: autoescapeMode,
		wr:         wr,
		context:    initialScope,
		ij:         t.ij,
	}
	defer state.errRecover(&err)
	state.walk(tmpl.Node)
	return
}
