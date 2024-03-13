package staticlint

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const usingExitInMainWarn = "using exit in main"

// ExitMainAnalyzer запрещает использовать прямой вызов os.Exit в функции main пакета main.
var ExitMainAnalyzer = &analysis.Analyzer{
	Name: "exitmain",
	Doc:  "check using exit in main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	const (
		mainPackageName = "main"
		mainFuncName    = "main"
		osPkgPath       = "os"
	)

	isExitCall := func(call *ast.CallExpr) bool {
		const exitFuncName = "Exit"
		switch x := call.Fun.(type) {
		case *ast.SelectorExpr:
			obj := pass.TypesInfo.Uses[x.Sel]
			if obj != nil && isFuncDef(obj, exitFuncName, osPkgPath) {
				return true
			}
		case *ast.Ident:
			obj := pass.TypesInfo.Uses[x]
			if obj != nil && isFuncDef(obj, exitFuncName, osPkgPath) {
				return true
			}
		}
		return false
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.File:
				if isIdent(x.Name, mainPackageName) {
					return true
				}
			case *ast.FuncDecl:
				if isIdent(x.Name, mainFuncName) {
					return true
				}
			case *ast.BlockStmt:
				return true
			case *ast.ExprStmt:
				return true
			case *ast.CallExpr:
				if isExitCall(x) {
					pass.Reportf(x.Pos(), usingExitInMainWarn)
				}
			}
			return false
		})
	}
	var res interface{} = nil
	return res, nil
}

func isIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

func isFuncDef(obj types.Object, objName, pkgPath string) bool {
	f, ok := obj.(*types.Func)
	return ok && f.Name() == objName && f.Pkg().Path() == pkgPath
}
