// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"fmt"
	"go/ast"

	tfslices "github.com/hashicorp/terraform-provider-aws/internal/slices"
	"golang.org/x/tools/go/packages"
)

type PackageFile struct {
	file *ast.File
}

type Package struct {
	name  string
	files []*PackageFile
}

func LoadPackage(sourcePackage string) (*Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax,
	}
	pkgs, err := packages.Load(cfg, sourcePackage)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", sourcePackage, err)
	}
	if len(pkgs) != 1 {
		return nil, fmt.Errorf("%d packages found", len(pkgs))
	}
	pkg := pkgs[0]

	return &Package{
		name: pkg.Name,
		files: tfslices.ApplyToAll(pkg.Syntax, func(file *ast.File) *PackageFile {
			return &PackageFile{
				file: file,
			}
		}),
	}, nil
}

func (pkg *Package) FindFunction(functionName string) *Function {
	for _, file := range pkg.files {
		if file.file != nil {
			for _, decl := range file.file.Decls {
				if funcDecl, ok := decl.(*ast.FuncDecl); ok {
					if funcDecl.Name.Name == functionName {
						return &Function{
							funcDecl: funcDecl,
						}
					}
				}
			}
		}
	}

	return nil
}

type Function struct {
	funcDecl *ast.FuncDecl
}

func (function *Function) Name() string {
	return function.funcDecl.Name.Name
}

func (function *Function) Params() []*ast.Field {
	return function.funcDecl.Type.Params.List
}

func (function *Function) Results() []*ast.Field {
	return function.funcDecl.Type.Results.List
}
