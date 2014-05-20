package contextual

import (
	"fmt"

	"github.com/robfig/soy/ast"

	"github.com/robfig/soy/template"
)

// Autoescape rewrites all templates in the given registry to add
// appropriately-escaping print directives to all print commands.
func Autoescape(reg template.Registry) (err error) {
	var currentTemplate string
	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("template %v: %v", currentTemplate, err2)
		}
	}()

	// See if the templates use contextual autoescaping
	var contextual = false
	for _, t := range reg.Templates {
		if t.Namespace.Autoescape == ast.AutoescapeContextual ||
			t.Autoescape == ast.AutoescapeContextual {
			contextual = true
			break
		}
	}

	if contextual {
		// if so, figure out the context at each point:
		// - beginning of a template
		// - beginning of a print tag

		// build a graph of template calls
		// assume the roots are all HTML context.
		var inferences engine
		var callGraph = newCallGraph(reg)
		for _, root := range callGraph.roots() {
			var startContext = context{state: statePCDATA}
			if root.Kind != "" {
				var ok bool
				startContext.state, ok = kindAttrToState[root.Kind]
				if !ok {
					panic("Kind " + root.Kind + " not recognized")
				}
			}
			engine.infer(root, startContext)
		}
	}

	// Apply the escaping
	for _, t := range reg.Templates {
		currentTemplate = t.Node.Name

		var a = simpleAutoescaper{t.Namespace.Autoescape}
		a.walk(t.Node)
	}
	return nil
}

type callGraph struct {
	registry        template.Registry
	calls           map[string]string // key calls value
	calledBy        map[string]string // key is called by value
	currentTemplate string
}

func newCallGraph(reg template.Registry) callGraph {
	var cgg = callGraph{reg, make(map[string]string), ""}
	for _, t := range reg.Templates {
		cgg.walk(t.Node)
	}
	return callGraph{cgg.calls}
}

func (g *callGraph) roots() []template.Template {
	var roots []template.Template
	for _, t := range g.registry.Templates {
		if _, ok := g.calledBy[t.Name]; !ok {
			roots = append(roots, t)
		}
	}
	return roots
}

func (g *callGraph) walk(node ast.Node) {
	switch node := node.(type) {
	case *ast.TemplateNode:
		g.currentTemplate = node.Name
	case *ast.CallNode:
		g.calls[g.currentTemplate] = node.Name
		g.calledBy[node.Name] = g.currentTemplate
	}
	if parent, ok := node.(ast.ParentNode); ok {
		for _, child := range parent.Children() {
			a.walk(child)
		}
	}
}

func makeCallGraph(reg template.Registry) map[string]string {

}
