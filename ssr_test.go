//go:build !wasm

package dom

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestRootCSS_NotEmpty(t *testing.T) {
	got := RootCSS()
	if got == "" {
		t.Error("RootCSS() returned an empty string")
	}
}

func TestRootCSS_ContainsRootSelector(t *testing.T) {
	got := RootCSS()
	if !strings.Contains(got, ":root") {
		t.Errorf("RootCSS() output does not contain ':root'\nGot:\n%s", got)
	}
}

func TestRootCSS_ContainsCoreToken(t *testing.T) {
	got := RootCSS()
	if !strings.Contains(got, "--mag-cua") {
		t.Errorf("RootCSS() output does not contain '--mag-cua'\nGot:\n%s", got)
	}
}

func TestRootCSS_ContainsDarkModeQuery(t *testing.T) {
	got := RootCSS()
	if !strings.Contains(got, "@media (prefers-color-scheme: dark)") {
		t.Errorf("RootCSS() output does not contain dark mode media query\nGot:\n%s", got)
	}
}

func TestRootCSS_DoesNotUseHighSpecificity(t *testing.T) {
	got := RootCSS()
	if strings.Contains(got, ":root:not([data-theme=\"light\"])") {
		t.Errorf("RootCSS() output contains high-specificity selector ':root:not([data-theme=\"light\"])'\nGot:\n%s", got)
	}
}

func TestRootCSS_AstShape(t *testing.T) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "ssr.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse ssr.go: %v", err)
	}

	var rootCSSFunc *ast.FuncDecl
	var rootCSSVar *ast.ValueSpec

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "RootCSS" {
			rootCSSFunc = fn
		}
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
			for _, spec := range gen.Specs {
				if v, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range v.Names {
						if name.Name == "rootCSS" {
							rootCSSVar = v
						}
					}
				}
			}
		}
	}

	if rootCSSFunc == nil {
		t.Fatal("could not find function RootCSS in ssr.go")
	}

	// Verify RootCSS returns rootCSS
	if len(rootCSSFunc.Body.List) != 1 {
		t.Fatalf("RootCSS function body should have exactly 1 statement, got %d", len(rootCSSFunc.Body.List))
	}
	ret, ok := rootCSSFunc.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatal("RootCSS function body does not end with a return statement")
	}
	if len(ret.Results) != 1 {
		t.Fatalf("RootCSS should return 1 value, got %d", len(ret.Results))
	}
	ident, ok := ret.Results[0].(*ast.Ident)
	if !ok || ident.Name != "rootCSS" {
		t.Fatalf("RootCSS should return 'rootCSS', got %v", ret.Results[0])
	}

	if rootCSSVar == nil {
		t.Fatal("could not find variable rootCSS in ssr.go")
	}

	// Find the GenDecl for rootCSSVar to check comments
	var rootCSSVarGenDecl *ast.GenDecl
	for _, decl := range node.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
			for _, spec := range gen.Specs {
				if spec == rootCSSVar {
					rootCSSVarGenDecl = gen
					break
				}
			}
		}
	}

	if rootCSSVarGenDecl == nil {
		t.Fatal("could not find GenDecl for rootCSS")
	}

	foundEmbed := false
	if rootCSSVarGenDecl.Doc != nil {
		for _, comment := range rootCSSVarGenDecl.Doc.List {
			if strings.HasPrefix(comment.Text, "//go:embed theme.css") {
				foundEmbed = true
				break
			}
		}
	}

	if !foundEmbed {
		t.Error("variable rootCSS in ssr.go is missing '//go:embed theme.css' directive")
	}
}
