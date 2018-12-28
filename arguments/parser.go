package arguments

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type Parser struct {
	currentWorkingDir string
	pathResolver      func(string, string) (string, error)
}

func NewParser(currentWorkingDir string) *Parser {
	return &Parser{
		currentWorkingDir: currentWorkingDir,
		pathResolver:      getSourcePath,
	}
}

func (argParser *Parser) ParseArguments(args ...string) (ParsedArguments, error) {
	if *packageFlag {
		return argParser.parsePackageArgs(args...)
	}
	return argParser.parseInterfaceArgs(args...)
}

func (argParser *Parser) parseInterfaceArgs(args ...string) (ParsedArguments, error) {
	var interfaceName string
	var outputPathFlagValue string
	var rootDestinationDir string
	var sourcePackageDir string
	var packagePath string

	if outputPathFlag != nil {
		outputPathFlagValue = *outputPathFlag
	}

	if len(args) > 1 {
		var err error
		sourcePackageDir, err = argParser.pathResolver(argParser.currentWorkingDir, args[0])
		if err != nil {
			return ParsedArguments{}, err
		}

		interfaceName = args[1]
		rootDestinationDir = sourcePackageDir
	} else {
		fullyQualifiedInterface := strings.Split(args[0], ".")
		interfaceName = fullyQualifiedInterface[len(fullyQualifiedInterface)-1]
		rootDestinationDir = argParser.currentWorkingDir
		packagePath = strings.Join(fullyQualifiedInterface[:len(fullyQualifiedInterface)-1], ".")
	}

	fakeImplName := getFakeName(interfaceName, *fakeNameFlag)

	outputPath := argParser.getOutputPath(
		rootDestinationDir,
		fakeImplName,
		outputPathFlagValue,
	)

	packageName := restrictToValidPackageName(filepath.Base(filepath.Dir(outputPath)))
	if packagePath == "" {
		packagePath = sourcePackageDir
	}
	if strings.HasPrefix(packagePath, build.Default.GOPATH) {
		packagePath = strings.Replace(packagePath, build.Default.GOPATH+"/src/", "", -1)
	}

	log.Printf("Parsed Arguments:\nInterface Name: %s\nPackage Path: %s\nDestination Package Name: %s", interfaceName, packagePath, packageName)
	return ParsedArguments{
		GenerateInterfaceAndShimFromPackageDirectory: false,
		OutputPath:  outputPath,
		PackagePath: packagePath,

		InterfaceName:          interfaceName,
		DestinationPackageName: packageName,
		FakeImplName:           fakeImplName,

		PrintToStdOut: any(args, "-"),
	}, nil
}

func (argParser *Parser) parsePackageArgs(args ...string) (ParsedArguments, error) {
	packagePath := args[0]
	packageName := path.Base(packagePath) + "shim"

	var outputPath string
	if *outputPathFlag != "" {
		// TODO: sensible checking of dirs and symlinks
		outputPath = *outputPathFlag
	} else {
		outputPath = path.Join(argParser.currentWorkingDir, packageName)
	}

	log.Printf("Parsed Arguments:\nPackage Name: %s\nDestination Package Name: %s", packagePath, packageName)
	return ParsedArguments{
		GenerateInterfaceAndShimFromPackageDirectory: true,
		OutputPath:             outputPath,
		PackagePath:            packagePath,
		DestinationPackageName: packageName,
		FakeImplName:           strings.ToUpper(path.Base(packagePath))[:1] + path.Base(packagePath)[1:],
		PrintToStdOut:          any(args, "-"),
	}, nil
}

type ParsedArguments struct {
	GenerateInterfaceAndShimFromPackageDirectory bool

	PackagePath string // package path to the package containing the interface to fake
	OutputPath  string // path to write the fake file to

	DestinationPackageName string // often the base-dir for OutputPath but must be a valid package name

	InterfaceName string // the interface to counterfeit
	FakeImplName  string // the name of the struct implementing the given interface

	PrintToStdOut bool
}

func fixupUnexportedNames(interfaceName string) string {
	asRunes := []rune(interfaceName)
	if len(asRunes) == 0 || !unicode.IsLower(asRunes[0]) {
		return interfaceName
	}
	asRunes[0] = unicode.ToUpper(asRunes[0])
	return string(asRunes)
}

func getFakeName(interfaceName, arg string) string {
	if arg == "" {
		interfaceName = fixupUnexportedNames(interfaceName)
		return "Fake" + interfaceName
	} else {
		return arg
	}
}

var camelRegexp = regexp.MustCompile("([a-z])([A-Z])")

func (argParser *Parser) getOutputPath(rootDestinationDir, fakeName, outputPathFlagValue string) string {
	if outputPathFlagValue == "" {
		snakeCaseName := strings.ToLower(camelRegexp.ReplaceAllString(fakeName, "${1}_${2}"))
		return filepath.Join(rootDestinationDir, packageNameForPath(rootDestinationDir), snakeCaseName+".go")
	} else {
		if !filepath.IsAbs(outputPathFlagValue) {
			outputPathFlagValue = filepath.Join(argParser.currentWorkingDir, outputPathFlagValue)
		}
		return outputPathFlagValue
	}
}

func packageNameForPath(pathToPackage string) string {
	_, packageName := filepath.Split(pathToPackage)
	return packageName + "fakes"
}

func getSourcePath(workingDir, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}

	evaluatedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("No such file/directory/package: '%s'", path)
	}

	stat, err := os.Stat(evaluatedPath)
	if err != nil {
		return "", fmt.Errorf("No such file/directory/package: '%s'", path)
	}

	if stat.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}

func any(slice []string, needle string) bool {
	for _, str := range slice {
		if str == needle {
			return true
		}
	}

	return false
}

func restrictToValidPackageName(input string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		} else {
			return -1
		}
	}, input)
}
