package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"log"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/imports"
)

// FakeMode indicates the type of Fake to generate.
type FakeMode int

// FakeMode can be Interface, Function, or Package.
const (
	InterfaceOrFunction FakeMode = iota
	Package
)

// Fake is used to generate a Fake implementation of an interface.
type Fake struct {
	Mode FakeMode

	// DestinationPackage name used in the package clause of the fake file.
	DestinationPackage string
	// Name of the fake interface or struct, used in the type declaration.
	Name string

	// Target is the type of the interface, or function to be faked.
	Target *types.TypeName
	// TargetName is the name of the interface, function, or package to be faked.
	TargetName string
	// TargetImport is the import used to reference the type to be faked.
	TargetImport Import

	Imports          Imports
	Methods          []Method
	Function         Method
	WorkingDirectory string
}

// Method is a method of the interface.
type Method struct {
	Name    string
	Params  Params
	Returns Returns
}

// NewFake returns a Fake that loads the package and finds the interface or the
// function.
func NewFake(fakeMode FakeMode, targetName string, packagePath string, fakeName string, destinationPackage string, workingDir string) (*Fake, error) {
	f := &Fake{
		TargetName:         targetName,
		Name:               fakeName,
		Mode:               fakeMode,
		DestinationPackage: destinationPackage,
		WorkingDirectory:   workingDir,
		Imports:            newImports(),
	}

	f.Imports.Add("sync", "sync")
	pkgs, err := f.loadPackages(packagePath)
	if err != nil {
		return nil, err
	}

	// TODO: Package mode here
	pkg, err := f.findPackage(pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to find %s: %v", packagePath, err)
	}

	switch {
	case f.IsInterface():
		f.Methods = f.loadMethods(interfaceMethodSet(f.Target.Type()))
	case f.Mode == Package:
		f.Methods = f.loadMethods(packageMethodSet(pkg))
	case f.IsFunction():
		if err := f.loadMethodForFunction(); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// IsInterface indicates whether the fake is for an interface.
func (f *Fake) IsInterface() bool {
	if f.Target == nil || f.Target.Type() == nil {
		return false
	}
	return types.IsInterface(f.Target.Type())
}

// IsFunction indicates whether the fake is for a function..
func (f *Fake) IsFunction() bool {
	if f.Target == nil || f.Target.Type() == nil || f.Target.Type().Underlying() == nil {
		return false
	}
	_, ok := f.Target.Type().Underlying().(*types.Signature)
	return ok
}

func unexport(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

func export(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func isExported(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsUpper(r)
}

// Generate uses the Fake to generate an implementation, optionally running
// goimports on the output.
func (f *Fake) Generate(runImports bool) ([]byte, error) {
	var tmpl *template.Template
	if f.IsInterface() {
		log.Printf("Writing fake %s for interface %s to package %s\n", f.Name, f.TargetName, f.DestinationPackage)
		tmpl = template.Must(template.New("fake").Funcs(interfaceFuncs).Parse(interfaceTemplate))
	}
	if f.IsFunction() {
		log.Printf("Writing fake %s for function %s to package %s\n", f.Name, f.TargetName, f.DestinationPackage)
		tmpl = template.Must(template.New("fake").Funcs(functionFuncs).Parse(functionTemplate))
	}
	if f.Mode == Package {
		log.Printf("Writing fake %s for package %s to package %s\n", f.Name, f.TargetImport.PkgPath, f.DestinationPackage)
		tmpl = template.Must(template.New("fake").Funcs(packageFuncs).Parse(packageTemplate))
	}
	if tmpl == nil {
		return nil, errors.New("counterfeiter can only generate fakes for interfaces or specific functions")
	}

	b := &bytes.Buffer{}
	tmpl.Execute(b, f)
	if runImports {
		return imports.Process("counterfeiter_temp_process_file", b.Bytes(), nil)
	}
	return b.Bytes(), nil
}
