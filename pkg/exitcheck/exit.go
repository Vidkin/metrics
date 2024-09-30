// Package exitcheck provides an analyzer that checks for the usage of
// os.Exit in the main function of the main package.
package exitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// ExitMainAnalyzer is an analyzer that checks for the usage of
// os.Exit in the main function of the main package.
var ExitMainAnalyzer = &analysis.Analyzer{
	Name: "exitmain",
	Doc:  "check for os.Exit in main.Main",
	Run:  run,
}

// run is the main analysis function for the ExitMainAnalyzer.
// It inspects the abstract syntax tree (AST) of Go source files
// to identify calls to os.Exit within the main function of the
// main package.
// Parameters:
//   - pass: An analysis.Pass that contains the files to be analyzed
//     and provides methods for reporting issues.
//
// Returns:
//   - An interface{} (always nil in this case) and an error (always nil).
func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		if f.Name.Name == "main" {
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					if x.Name.Name == "main" {
						for _, v := range x.Body.List {
							if ex, ok := v.(*ast.ExprStmt); ok {
								if call, ok := ex.X.(*ast.CallExpr); ok {
									if sl, ok := call.Fun.(*ast.SelectorExpr); ok {
										if id, ok := sl.X.(*ast.Ident); ok {
											if id.Name == "os" && sl.Sel.Name == "Exit" {
												pass.Reportf(x.Pos(), "exit in main func of main package")
											}
										}
									}
								}
							}
						}

					}
				}
				return true
			})
		}
	}
	return nil, nil
}
