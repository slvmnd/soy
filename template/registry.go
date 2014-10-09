// Package template provides convenient access to groups of parsed soy files.
package template

import (
	"fmt"
	"log"
	"strings"

	"github.com/slvmnd/soy/ast"
)

type delPair struct {
	name    string
	variant string
}

// Registry provides convenient access to a collection of parsed Soy templates.
type Registry struct {
	SoyFiles     []*ast.SoyFileNode
	Templates    []Template
	DelTemplates []DelTemplate

	// sourceByTemplateName maps FQ template name to the input source it came from.
	sourceByTemplateName    map[string]string
	sourceByDelTemplateName map[delPair]string
}

// Add the given soy file node (and all contained templates) to this registry.
func (r *Registry) Add(soyfile *ast.SoyFileNode) error {
	if r.sourceByTemplateName == nil {
		r.sourceByTemplateName = make(map[string]string)
	}
	var ns *ast.NamespaceNode
	for _, node := range soyfile.Body {
		switch node := node.(type) {
		case *ast.SoyDocNode:
			continue
		case *ast.NamespaceNode:
			ns = node
		default:
			return fmt.Errorf("expected namespace, found %v", node)
		}
		break
	}
	if ns == nil {
		return fmt.Errorf("namespace required")
	}

	r.SoyFiles = append(r.SoyFiles, soyfile)
	for i := 0; i < len(soyfile.Body); i++ {
		var tn, ok = soyfile.Body[i].(*ast.TemplateNode)
		if !ok {
			var dtn, ok = soyfile.Body[i].(*ast.DelTemplateNode)
			if !ok {
				continue
			}

			// Technically every template requires soydoc, but having to add empty
			// soydoc just to get a template to compile is just stupid.  (There is a
			// separate data ref check to ensure any variables used are declared as
			// params, anyway).
			sdn, ok := soyfile.Body[i-1].(*ast.SoyDocNode)
			if !ok {
				sdn = &ast.SoyDocNode{dtn.Pos, nil}
			}
			r.DelTemplates = append(r.DelTemplates, DelTemplate{sdn, dtn, ns})
			r.sourceByDelTemplateName[delPair{dtn.Name, dtn.Variant}] = soyfile.Text
			continue
		}

		// Technically every template requires soydoc, but having to add empty
		// soydoc just to get a template to compile is just stupid.  (There is a
		// separate data ref check to ensure any variables used are declared as
		// params, anyway).
		sdn, ok := soyfile.Body[i-1].(*ast.SoyDocNode)
		if !ok {
			sdn = &ast.SoyDocNode{tn.Pos, nil}
		}
		r.Templates = append(r.Templates, Template{sdn, tn, ns})
		r.sourceByTemplateName[tn.Name] = soyfile.Text
	}
	return nil
}

// Template allows lookup by (fully-qualified) template name.
// The resulting template is returned and a boolean indicating if it was found.
func (r *Registry) Template(name string) (Template, bool) {
	for _, t := range r.Templates {
		if t.Node.Name == name {
			return t, true
		}
	}
	return Template{}, false
}

// LineNumber computes the line number in the input source for the given node
// within the given template.
func (r *Registry) LineNumber(templateName string, node ast.Node) int {
	var src, ok = r.sourceByTemplateName[templateName]
	if !ok {
		log.Println("template not found:", templateName)
		return 0
	}
	return 1 + strings.Count(src[:node.Position()], "\n")
}

// DelTemplate allows lookup by (fully-qualified) template name and variant argument.
// The resulting template is returned and a boolean indicating if it was found.
func (r *Registry) DelTemplate(name, variant string) (DelTemplate, bool) {
	for _, t := range r.DelTemplates {
		if t.Node.Name == name && t.Node.Variant == variant {
			return t, true
		}
	}
	return DelTemplate{}, false
}

// DelLineNumber computes the line number in the input source for the given node
// within the given template variant.
func (r *Registry) DelLineNumber(templateName string, node ast.Node) int {
	var src, ok = r.sourceByTemplateName[templateName]
	if !ok {
		log.Println("template not found:", templateName)
		return 0
	}
	return 1 + strings.Count(src[:node.Position()], "\n")
}
