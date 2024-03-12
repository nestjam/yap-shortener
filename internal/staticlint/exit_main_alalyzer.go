package staticlint

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

const usingExitInMainWarn = "using exit in main"

var ExitMainAnalyzer = &analysis.Analyzer{
	Name: "exitmain",
	Doc:  "check using exit in main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	const (
		mainPackageName      = "main"
		mainFuncName         = "main"
		osPackageDefaultName = "os"
	)

	isExitCall := func(call *ast.CallExpr, osPackageName string) bool {
		const exitFuncName = "Exit"
		switch x := call.Fun.(type) {
		case *ast.SelectorExpr:
			if isIdent(x.Sel, exitFuncName) && isPackageIdent(x.X, osPackageName) {
				return true
			}
		case *ast.Ident:
			if isIdent(x, exitFuncName) && osPackageName == "." {
				return true
			}
		}
		return false
	}

	for _, file := range pass.Files {
		isOSImported := false
		osPackageName := osPackageDefaultName

		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.File:
				if isIdent(x.Name, mainPackageName) {
					return true
				}
			case *ast.GenDecl:
				if x.Tok == token.IMPORT {
					return true
				}
			case *ast.ImportSpec:
				if x.Path.Value == "\"os\"" {
					isOSImported = true
					if x.Name != nil {
						osPackageName = x.Name.Name
					}
				}
			}
			return false
		})

		if !isOSImported {
			continue
		}

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
				if isExitCall(x, osPackageName) {
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

func isPackageIdent(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Obj == nil && id.Name == name
}
