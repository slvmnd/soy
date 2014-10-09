// Package parsepasses contains routines that validate or rewrite a Soy AST.
package parsepasses

import (
	"fmt"

	"github.com/slvmnd/soy/ast"
	"github.com/slvmnd/soy/template"
)

// CheckDataRefs validates that:
//  1. all data refs are provided by @params or {let} nodes (except $ij)
//  2. any data declared as a @param is used by the template (or passed via {call})
//  3. all {call} params are declared as @params in the called template soydoc.
//  4. a {call}'ed template is passed all required @params, or a data="$var"
//  5. {call}'d templates actually exist in the registry.
//  6. any variable created by {let} is used somewhere
//  7. {let} variable names are valid.  ('ij' is not allowed.)
func CheckDataRefs(reg template.Registry) (err error) {
	var currentTemplate string
	defer func() {
		if err2 := recover(); err2 != nil {
			err = fmt.Errorf("template %v: %v", currentTemplate, err2)
		}
	}()

	for _, t := range reg.Templates {
		currentTemplate = t.Node.Name
		tc := newTemplateChecker(reg, t.Doc.Params)
		tc.checkTemplate(t.Node.Body)

		// check that all params appear in the usedKeys
		for _, param := range tc.params {
			if !contains(tc.usedKeys, param) {
				panic(fmt.Errorf("param %q is unused", param))
			}
		}
	}
	return nil
}

type templateChecker struct {
	registry template.Registry
	params   []string
	letVars  []string
	forVars  []string
	usedKeys []string
}

func newTemplateChecker(reg template.Registry, params []*ast.SoyDocParamNode) *templateChecker {
	var paramNames []string
	for _, param := range params {
		paramNames = append(paramNames, param.Name)
	}
	return &templateChecker{reg, paramNames, nil, nil, nil}
}

func (tc *templateChecker) checkTemplate(node ast.Node) {
	switch node := node.(type) {
	case *ast.LetValueNode:
		tc.checkLet(node.Name)
		tc.letVars = append(tc.letVars, node.Name)
	case *ast.LetContentNode:
		tc.checkLet(node.Name)
		tc.letVars = append(tc.letVars, node.Name)
	case *ast.CallNode:
		tc.checkCall(node)
	case *ast.DelCallNode:
		tc.checkDelCall(node)
	case *ast.ForNode:
		tc.forVars = append(tc.forVars, node.Var)
	case *ast.DataRefNode:
		tc.visitKey(node.Key)
	}
	if parent, ok := node.(ast.ParentNode); ok {
		tc.recurse(parent)
	}
}

// checkLet ensures that the let variable has an allowed name.
func (tc *templateChecker) checkLet(varName string) {
	if varName == "ij" {
		panic("Invalid variable name in 'let' command text: '$ij'")
	}
}

func (tc *templateChecker) checkCall(node *ast.CallNode) {
	var callee, ok = tc.registry.Template(node.Name)
	if !ok {
		panic(fmt.Errorf("{call}: template %q not found", node.Name))
	}

	tc.checkCallAgainstDoc(node, callee.Doc)
}

func (tc *templateChecker) checkDelCall(node *ast.DelCallNode) {
	var callee, ok = tc.registry.DelTemplate(node.Name, node.Variant.String())
	if !ok {
		panic(fmt.Errorf("{delcall}: template %q with variant %q not found", node.Name, node.Variant.String()))
	}

	tc.checkCallAgainstDoc(&node.CallNode, callee.Doc)
}

func (tc *templateChecker) checkCallAgainstDoc(node *ast.CallNode, doc *ast.SoyDocNode) {
	// collect callee's list of required/allowed params
	var allCalleeParamNames, requiredCalleeParamNames []string
	for _, param := range doc.Params {
		allCalleeParamNames = append(allCalleeParamNames, param.Name)
		if !param.Optional {
			requiredCalleeParamNames = append(requiredCalleeParamNames, param.Name)
		}
	}

	// collect caller's list of params.
	// if {call} passes data="all", expand that into all of the key names that
	// the caller has in common with params of the callee.
	var callerParamNames []string
	if node.AllData {
		for _, param := range tc.params {
			if contains(allCalleeParamNames, param) {
				tc.usedKeys = append(tc.usedKeys, param)
				callerParamNames = append(callerParamNames, param)
			}
		}
	}
	// add the {param}'s
	for _, callParam := range node.Params {
		switch callParam := callParam.(type) {
		case *ast.CallParamValueNode:
			callerParamNames = append(callerParamNames, callParam.Key)
		case *ast.CallParamContentNode:
			callerParamNames = append(callerParamNames, callParam.Key)
		default:
			panic("unexpected call param type")
		}
	}

	// reconcile the two param lists.
	// check: all {call} params are declared as @params in the called template soydoc.
	for _, callParamName := range callerParamNames {
		if !contains(allCalleeParamNames, callParamName) {
			panic(fmt.Errorf("Param %q is not declared by the callee.", callParamName))
		}
	}

	// check: a {call}'ed template is passed all required @params, or a data="$var"
	if node.Data != nil {
		return
	}
	for _, requiredCalleeParam := range requiredCalleeParamNames {
		if !contains(callerParamNames, requiredCalleeParam) {
			panic(fmt.Errorf("Required param %q is not passed by the call: %v",
				requiredCalleeParam, node))
		}
	}
}

func (tc *templateChecker) recurse(parent ast.ParentNode) {
	var initialForVars = len(tc.forVars)
	var initialLetVars = len(tc.letVars)
	var initialUsedKeys = len(tc.usedKeys)
	for _, child := range parent.Children() {
		tc.checkTemplate(child)
	}
	tc.forVars = tc.forVars[:initialForVars]

	// quick return if there were no {let}s
	if initialLetVars == len(tc.letVars) {
		return
	}

	// "pop" the {let} variables, as well as their usages.
	// (this is necessary to handle shadowing of @params by {let} vars)
	var letVarsGoingOutOfScope = tc.letVars[initialLetVars:]
	var usedKeysToKeep, usedLets []string
	for _, key := range tc.usedKeys[initialUsedKeys:] {
		if contains(letVarsGoingOutOfScope, key) {
			usedLets = append(usedLets, key)
		} else {
			usedKeysToKeep = append(usedKeysToKeep, key)
		}
	}

	// check that any let variables leaving scope have been used
	for _, letVar := range letVarsGoingOutOfScope {
		if !contains(usedLets, letVar) {
			panic(fmt.Errorf("{let} variable %q is not used.", letVar))
		}
	}

	tc.usedKeys = append(tc.usedKeys[:initialUsedKeys], usedKeysToKeep...)
	tc.letVars = tc.letVars[:initialLetVars]
}

func (tc *templateChecker) visitKey(key string) {
	// record that this key was used in the template.
	tc.usedKeys = append(tc.usedKeys, key)

	// check that the key was provided by a @param or {let}
	if !tc.checkKey(key) {
		panic(fmt.Errorf("data ref %q not found. params: %v, let variables: %v",
			key, tc.params, tc.letVars))
	}
}

// checkKey returns true if the given key exists as a param or {let} variable.
func (tc *templateChecker) checkKey(key string) bool {
	if key == "ij" {
		return true
	}
	for _, param := range tc.params {
		if param == key {
			return true
		}
	}
	for _, varName := range tc.letVars {
		if varName == key {
			return true
		}
	}
	for _, varName := range tc.forVars {
		if varName == key {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, candidate := range slice {
		if candidate == item {
			return true
		}
	}
	return false
}
