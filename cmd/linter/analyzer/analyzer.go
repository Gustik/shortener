package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "reports panic, log.Fatal*, and os.Exit calls outside main() of main package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	isMainPkg := pass.Pkg.Name() == "main"
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "panic" {
					pass.Reportf(call.Pos(), "use of panic is forbidden")
				}
			case *ast.SelectorExpr:
				pkgIdent, ok := fn.X.(*ast.Ident)
				if !ok {
					return true
				}
				pkg, fun := pkgIdent.Name, fn.Sel.Name
				if pkg == "log" && (fun == "Fatal" || fun == "Fatalf" || fun == "Fatalln") {
					if !isMainPkg || !isInsideMainFunc(file, call.Pos()) {
						pass.Reportf(call.Pos(), "log.%s is forbidden outside main() of main package", fun)
					}
				}
				if pkg == "os" && fun == "Exit" {
					if !isMainPkg || !isInsideMainFunc(file, call.Pos()) {
						pass.Reportf(call.Pos(), "os.Exit is forbidden outside main() of main package")
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func isInsideMainFunc(file *ast.File, pos token.Pos) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Body == nil {
			continue
		}
		if fn.Body.Lbrace <= pos && pos <= fn.Body.Rbrace {
			return true
		}
	}
	return false
}
