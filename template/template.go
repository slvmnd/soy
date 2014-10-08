package template

import "github.com/slvmnd/soy/ast"

// Template is a Soy template's parse tree, including the relevant context
// (preceeding soydoc and namespace).
type Template struct {
	Doc       *ast.SoyDocNode    // this template's SoyDoc
	Node      *ast.TemplateNode  // this template's node
	Namespace *ast.NamespaceNode // this template's namespace
}
