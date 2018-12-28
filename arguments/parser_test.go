package arguments

import (
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestParsingArguments(t *testing.T) {
	spec.Run(t, "ParsingArguments", testParsingArguments, spec.Report(report.Terminal{}))
}

func testParsingArguments(t *testing.T, when spec.G, it spec.S) {
	var parsedArgs ParsedArguments
	var args []string
	var cwd string

	justBefore := func() {
		parser := NewParser(cwd)
		parser.pathResolver = func(cwd string, path string) (string, error) {
			return filepath.Join(cwd, path), nil
		}
		var err error
		parsedArgs, err = parser.ParseArguments(args...)
		Expect(err).To(BeNil())
	}

	it.Before(func() {
		RegisterTestingT(t)
		*packageFlag = false
		*outputPathFlag = ""
		cwd = "/home/test-user/workspace"
	})

	when("when the -p flag is provided", func() {
		it.Before(func() {
			args = []string{"os"}
			*packageFlag = true
			justBefore()
		})

		it("doesn't parse extraneous arguments", func() {
			Expect(parsedArgs.InterfaceName).To(Equal(""))
			Expect(parsedArgs.FakeImplName).To(Equal("Os"))
		})

		when("given a stdlib package", func() {
			it("sets arguments as expected", func() {
				Expect(parsedArgs.OutputPath).To(Equal(path.Join(cwd, "osshim")))
				Expect(parsedArgs.DestinationPackageName).To(Equal("osshim"))
			})
		})

		when("given a relative path to a path to a package", func() {})
	})

	when("when a single argument is provided", func() {
		it.Before(func() {
			args = []string{"someonesinterfaces.AnInterface"}
			justBefore()
		})

		it("indicates to not print to stdout", func() {
			Expect(parsedArgs.PrintToStdOut).To(BeFalse())
		})

		it("provides a name for the fake implementing the interface", func() {
			Expect(parsedArgs.FakeImplName).To(Equal("FakeAnInterface"))
		})

		it("provides a path for the interface source", func() {
			Expect(parsedArgs.PackagePath).To(Equal("someonesinterfaces"))
		})

		it("treats the last segment as the interface to counterfeit", func() {
			Expect(parsedArgs.InterfaceName).To(Equal("AnInterface"))
		})

		it("snake cases the filename for the output directory", func() {
			Expect(filepath.Base(parsedArgs.OutputPath)).To(Equal("fake_an_interface.go"))
		})
	})

	when("when a single argument is provided with the output directory", func() {
		it.Before(func() {
			*outputPathFlag = "/tmp/foo"
			args = []string{"io.Writer"}
			justBefore()
		})

		it("indicates to not print to stdout", func() {
			Expect(parsedArgs.PrintToStdOut).To(BeFalse())
		})

		it("provides a name for the fake implementing the interface", func() {
			Expect(parsedArgs.FakeImplName).To(Equal("FakeWriter"))
		})

		it("provides a path for the interface source", func() {
			Expect(parsedArgs.PackagePath).To(Equal("io"))
		})

		it("treats the last segment as the interface to counterfeit", func() {
			Expect(parsedArgs.InterfaceName).To(Equal("Writer"))
		})

		it("copies the provided output path into the result", func() {
			Expect(parsedArgs.OutputPath).To(Equal("/tmp/foo"))
		})
	})

	when("when two arguments are provided", func() {
		it.Before(func() {
			args = []string{"my/my5package", "MySpecialInterface"}
			justBefore()
		})

		it("indicates to not print to stdout", func() {
			Expect(parsedArgs.PrintToStdOut).To(BeFalse())
		})

		it("provides a name for the fake implementing the interface", func() {
			Expect(parsedArgs.FakeImplName).To(Equal("FakeMySpecialInterface"))
		})

		it("treats the second argument as the interface to counterfeit", func() {
			Expect(parsedArgs.InterfaceName).To(Equal("MySpecialInterface"))
		})

		it("snake cases the filename for the output directory", func() {
			Expect(filepath.Base(parsedArgs.OutputPath)).To(Equal("fake_my_special_interface.go"))
		})

		it("specifies the destination package name", func() {
			Expect(parsedArgs.DestinationPackageName).To(Equal("my5packagefakes"))
		})

		when("when the interface is unexported", func() {
			it.Before(func() {
				args = []string{"my/mypackage", "mySpecialInterface"}
				justBefore()
			})

			it("fixes up the fake name to be TitleCase", func() {
				Expect(parsedArgs.FakeImplName).To(Equal("FakeMySpecialInterface"))
			})

			it("snake cases the filename for the output directory", func() {
				Expect(filepath.Base(parsedArgs.OutputPath)).To(Equal("fake_my_special_interface.go"))
			})
		})
	})

	when("when the output dir contains characters inappropriate for a package name", func() {
		it.Before(func() {
			args = []string{"@my-special-package[]{}", "MySpecialInterface"}
			justBefore()
		})

		it("should choose a valid package name", func() {
			Expect(parsedArgs.DestinationPackageName).To(Equal("myspecialpackagefakes"))
		})
	})

	when("when three arguments are provided", func() {
		when("and the third one is '-'", func() {
			it.Before(func() {
				args = []string{"my/mypackage", "MySpecialInterface", "-"}
				justBefore()
			})

			it("treats the second argument as the interface to counterfeit", func() {
				Expect(parsedArgs.InterfaceName).To(Equal("MySpecialInterface"))
			})

			it("provides a name for the fake implementing the interface", func() {
				Expect(parsedArgs.FakeImplName).To(Equal("FakeMySpecialInterface"))
			})

			it("indicates that the fake should be printed to stdout", func() {
				Expect(parsedArgs.PrintToStdOut).To(BeTrue())
			})

			it("snake cases the filename for the output directory", func() {
				Expect(filepath.Base(parsedArgs.OutputPath)).To(Equal("fake_my_special_interface.go"))
			})
		})

		when("and the third one is some random input", func() {
			it.Before(func() {
				args = []string{"my/mypackage", "MySpecialInterface", "WHOOPS"}
				justBefore()
			})

			it("indicates to not print to stdout", func() {
				Expect(parsedArgs.PrintToStdOut).To(BeFalse())
			})
		})
	})

	when("when the output dir contains underscores in package name", func() {
		it.Before(func() {
			args = []string{"fake_command_runner", "MySpecialInterface"}
			justBefore()
		})

		it("should ensure underscores are in the package name", func() {
			Expect(parsedArgs.DestinationPackageName).To(Equal("fake_command_runnerfakes"))
		})
	})
}
