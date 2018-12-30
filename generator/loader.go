package generator

import (
	"fmt"
	"go/types"
	"log"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

func (f *Fake) loadPackages(pkgPath string) ([]*packages.Package, error) {
	log.Println("loading packages...")
	p, err := packages.Load(&packages.Config{
		Mode:  packages.LoadSyntax,
		Dir:   f.WorkingDirectory,
		Tests: true,
	}, pkgPath)
	if err != nil {
		return nil, err
	}
	for i := range p {
		if len(p[i].Errors) > 0 {
			if i == 0 {
				err = p[i].Errors[0]
			}
			for j := range p[i].Errors {
				log.Printf("error loading packages: %v", strings.TrimPrefix(fmt.Sprintf("%v", p[i].Errors[j]), "-: "))
			}
		}
	}
	if err != nil {
		return nil, err
	}
	log.Printf("loaded %v packages\n", len(p))
	return p, nil
}

func (f *Fake) findPackage(pkgs []*packages.Package) (*packages.Package, error) {
	var target *types.TypeName
	var pkg *packages.Package
	for i := range pkgs {
		if pkgs[i].Types == nil || pkgs[i].Types.Scope() == nil {
			continue
		}
		pkg = pkgs[i]
		if f.Mode == Package {
			break
		}

		obj := pkg.Types.Scope().Lookup(f.TargetName)
		if obj != nil {
			if typeName, ok := obj.(*types.TypeName); ok {
				target = typeName
				break
			}
		}
		pkg = nil
	}
	if pkg == nil {
		switch f.Mode {
		case Package:
			return nil, fmt.Errorf("cannot find package")
		case InterfaceOrFunction:
			return nil, fmt.Errorf("cannot find package with target: %s", f.TargetName)
		}
	}
	return pkg, f.loadPackage(pkg, target)
}

func (f *Fake) loadPackage(pkg *packages.Package, target *types.TypeName) error {
	f.Target = target
	f.TargetImport = f.Imports.Add(pkg.Name, imports.VendorlessPath(pkg.PkgPath))
	if f.Mode != Package {
		f.TargetName = target.Name()
	}

	if f.Mode == InterfaceOrFunction {
		if !f.IsInterface() && !f.IsFunction() {
			return fmt.Errorf("cannot generate an fake for %s because it is not an interface or function", f.TargetName)
		}
	}

	switch {
	case f.IsInterface():
		log.Printf("Found interface with name: [%s]\n", f.TargetName)
	case f.IsFunction():
		log.Printf("Found function with name: [%s]\n", f.TargetName)
	case f.Mode == Package:
		log.Printf("Found package with name: [%s]\n", f.TargetImport.PkgPath)
	}
	return nil
}
