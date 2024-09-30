package analysis

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ExitMainAnalyzer = &analysis.Analyzer{
	Name: "exitmain",
	Doc:  "check for os.Exit in main.Main",
	Run:  run,
}

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
