package generator

import (
	"fmt"
	"go/types"
	"log"
	"path"
	"reflect"
	"strings"

	"golang.org/x/tools/imports"
)

// Imports indexes imports by package path and alias so that all imports have a
// unique alias, and no package is included twice.
type Imports struct {
	ByAlias   map[string]Import
	ByPkgPath map[string]Import
}

func newImports() Imports {
	return Imports{
		ByAlias:   make(map[string]Import),
		ByPkgPath: make(map[string]Import),
	}
}

// Import is a package import with the associated alias for that package.
type Import struct {
	Alias   string
	PkgPath string
}

// String returns a string that may be used as an import line in a go source
// file. Imports with aliases that match the package basename are printed without
// an alias.
func (i Import) String() string {
	if path.Base(i.PkgPath) == i.Alias {
		return `"` + i.PkgPath + `"`
	}
	return fmt.Sprintf(`%s "%s"`, i.Alias, i.PkgPath)
}

// AddImport creates an import with the given alias and path, and adds it to
// Fake.Imports.
func (i *Imports) Add(alias string, path string) Import {
	// TODO: why is there extra whitespace on these args?
	path = imports.VendorlessPath(strings.TrimSpace(path))
	alias = strings.TrimSpace(alias)

	imp, exists := i.ByPkgPath[path]
	if exists {
		return imp
	}

	imp, exists = i.ByAlias[alias]
	if exists {
		alias = uniqueAliasForImport(alias, i.ByAlias)
	}

	result := Import{Alias: alias, PkgPath: path}
	i.ByPkgPath[path] = result
	i.ByAlias[alias] = result
	return result
}

func uniqueAliasForImport(alias string, imports map[string]Import) string {
	for i := 0; ; i++ {
		newAlias := alias + string('a'+byte(i))
		if _, exists := imports[newAlias]; !exists {
			return newAlias
		}
	}
}

// AliasForPackage returns a package alias for the package.
func (i *Imports) AliasForPackage(p *types.Package) string {
	return i.ByPkgPath[imports.VendorlessPath(p.Path())].Alias
}

// addFromType inspects the given type and adds imports to the fake if importable
// types are found.
func (i *Imports) addFromType(typ types.Type) {
	if typ == nil {
		return
	}

	switch t := typ.(type) {
	case *types.Basic:
		return
	case *types.Pointer:
		i.addFromType(t.Elem())
	case *types.Map:
		i.addFromType(t.Key())
		i.addFromType(t.Elem())
	case *types.Chan:
		i.addFromType(t.Elem())
	case *types.Named:
		if t.Obj() != nil && t.Obj().Pkg() != nil {
			i.Add(t.Obj().Pkg().Name(), t.Obj().Pkg().Path())
		}
	case *types.Slice:
		i.addFromType(t.Elem())
	case *types.Array:
		i.addFromType(t.Elem())
	case *types.Interface:
		return
	case *types.Signature:
		i.addFromMethodSignature(t)
	default:
		log.Printf("!!! WARNING: Missing case for type %s\n", reflect.TypeOf(typ).String())
	}
}

func (i *Imports) addFromMethodSignature(sig *types.Signature) {
	for n := 0; n < sig.Results().Len(); n++ {
		ret := sig.Results().At(n)
		i.addFromType(ret.Type())
	}
	for n := 0; n < sig.Params().Len(); n++ {
		param := sig.Params().At(n)
		i.addFromType(param.Type())
	}
}
