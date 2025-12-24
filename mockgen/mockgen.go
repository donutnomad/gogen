// Copyright 2010 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// MockGen generates mock implementations of Go interfaces.
package mockgen

// TODO: This does not support recursive embedded interfaces.
// TODO: This does not support embedding package-local interfaces in a separate file.

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"go.uber.org/mock/mockgen/model"
)

// Options contains all configuration options for mock generation.
type Options struct {
	// Source is the input Go source file (source mode).
	// If non-empty, enables source mode.
	Source string

	// PackageName is the import path of the package (package mode).
	// Use "." to refer to the current path's package.
	PackageName string

	// Interfaces is a comma-separated list of interface names to mock (package mode).
	Interfaces []string

	// Destination is the output file path.
	// If empty, defaults to stdout.
	Destination string

	// MockNames is comma-separated interfaceName=mockName pairs of explicit mock names to use.
	// Mock names default to 'Mock' + interfaceName suffix.
	MockNames string

	// Package is the package name of the generated code.
	// Defaults to the package of the input with a 'mock_' prefix.
	Package string

	// SelfPackage is the full package import path for the generated code.
	// The purpose of this flag is to prevent import cycles in the generated code
	// by trying to include its own package. This can happen if the mock's package
	// is set to one of its inputs (usually the main one) and the output is stdio
	// so mockgen cannot detect the final output package.
	SelfPackage string

	// WritePkgComment writes package documentation comment (godoc) if true.
	WritePkgComment bool

	// WriteSourceComment writes original file (source mode) or interface names
	// (package mode) comment if true.
	WriteSourceComment bool

	// WriteGenerateDirective adds //go:generate directive to regenerate the mock.
	WriteGenerateDirective bool

	// BuildConstraint is added as //go:build <constraint> if non-empty.
	BuildConstraint string

	// Typed generates type-safe 'Return', 'Do', 'DoAndReturn' functions.
	Typed bool

	// Imports is comma-separated name=path pairs of explicit imports to use (source mode).
	Imports string

	// AuxFiles is comma-separated pkg=path pairs of auxiliary Go source files (source mode).
	AuxFiles string

	// ExcludeInterfaces is comma-separated names of interfaces to be excluded.
	ExcludeInterfaces string
}

// DefaultOptions returns Options with default values.
func DefaultOptions() Options {
	return Options{
		WritePkgComment:    true,
		WriteSourceComment: true,
	}
}

// Run generates mock implementations based on the provided options.
func Run(opts Options) error {
	var pkg *model.Package
	var err error

	// Switch between modes
	switch {
	case opts.Source != "": // source mode
		pkg, err = sourceModeWithOptions(SourceModeOptions{
			Source:            opts.Source,
			Imports:           opts.Imports,
			AuxFiles:          opts.AuxFiles,
			ExcludeInterfaces: opts.ExcludeInterfaces,
		})
	default: // package mode
		packageName := opts.PackageName
		if packageName == "" {
			return fmt.Errorf("package name is required in package mode")
		}
		if len(opts.Interfaces) == 0 {
			return fmt.Errorf("at least one interface is required")
		}

		if packageName == "." {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get current directory failed: %v", err)
			}
			packageName, err = packageNameOfDir(dir)
			if err != nil {
				return fmt.Errorf("parse package name failed: %v", err)
			}
		}
		parser := packageModeParser{}
		pkg, err = parser.parsePackage(packageName, opts.Interfaces)
	}

	if err != nil {
		return fmt.Errorf("loading input failed: %v", err)
	}

	outputPackageName := opts.Package
	if outputPackageName == "" {
		// pkg.Name in package mode is the base name of the import path,
		// which might have characters that are illegal to have in package names.
		outputPackageName = "mock_" + sanitize(pkg.Name)
	}

	// outputPackagePath represents the fully qualified name of the package of
	// the generated code. Its purposes are to prevent the module from importing
	// itself and to prevent qualifying type names that come from its own
	// package (i.e. if there is a type called X then we want to print "X" not
	// "package.X" since "package" is this package). This can happen if the mock
	// is output into an already existing package.
	outputPackagePath := opts.SelfPackage
	if outputPackagePath == "" && opts.Destination != "" {
		dstPath, err := filepath.Abs(filepath.Dir(opts.Destination))
		if err == nil {
			pkgPath, err := parsePackageImport(dstPath)
			if err == nil {
				outputPackagePath = pkgPath
			}
		}
	}

	g := &generator{
		buildConstraint:        opts.BuildConstraint,
		writePkgComment:        opts.WritePkgComment,
		writeSourceComment:     opts.WriteSourceComment,
		writeGenerateDirective: opts.WriteGenerateDirective,
		typed:                  opts.Typed,
		imports:                opts.Imports,
		selfPackage:            opts.SelfPackage,
	}
	if opts.Source != "" {
		g.filename = opts.Source
	} else {
		g.srcPackage = opts.PackageName
		g.srcInterfaces = strings.Join(opts.Interfaces, ",")
	}
	g.destination = opts.Destination

	if opts.MockNames != "" {
		g.mockNames = parseMockNames(opts.MockNames)
	}
	if err := g.Generate(pkg, outputPackageName, outputPackagePath); err != nil {
		return fmt.Errorf("failed generating mock: %v", err)
	}
	output := g.Output(true)
	dst := os.Stdout
	if len(opts.Destination) > 0 {
		if err := os.MkdirAll(filepath.Dir(opts.Destination), os.ModePerm); err != nil {
			return fmt.Errorf("unable to create directory: %v", err)
		}
		existing, err := os.ReadFile(opts.Destination)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed reading pre-existing destination file: %v", err)
		}
		if len(existing) == len(output) && bytes.Equal(existing, output) {
			return nil
		}
		f, err := os.Create(opts.Destination)
		if err != nil {
			return fmt.Errorf("failed opening destination file: %v", err)
		}
		defer f.Close()
		dst = f
	}
	if _, err := dst.Write(output); err != nil {
		return fmt.Errorf("failed writing to destination: %v", err)
	}
	return nil
}

func parseMockNames(names string) map[string]string {
	mocksMap := make(map[string]string)
	for _, kv := range strings.Split(names, ",") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 || parts[1] == "" {
			log.Fatalf("bad mock names spec: %v", kv)
		}
		mocksMap[parts[0]] = parts[1]
	}
	return mocksMap
}

func parseExcludeInterfaces(names string) map[string]struct{} {
	splitNames := strings.Split(names, ",")
	namesSet := make(map[string]struct{}, len(splitNames))
	for _, name := range splitNames {
		if name == "" {
			continue
		}

		namesSet[name] = struct{}{}
	}

	if len(namesSet) == 0 {
		return nil
	}

	return namesSet
}

// parseImportPackage get package import path via source file
// an alternative implementation is to use:
// cfg := &packages.Config{Mode: packages.NeedName, Tests: true, Dir: srcDir}
// pkgs, err := packages.Load(cfg, "file="+source)
// However, it will call "go list" and slow down the performance
func parsePackageImport(srcDir string) (string, error) {
	moduleMode := os.Getenv("GO111MODULE")
	// trying to find the module
	if moduleMode != "off" {
		currentDir := srcDir
		for {
			dat, err := os.ReadFile(filepath.Join(currentDir, "go.mod"))
			if os.IsNotExist(err) {
				if currentDir == filepath.Dir(currentDir) {
					// at the root
					break
				}
				currentDir = filepath.Dir(currentDir)
				continue
			} else if err != nil {
				return "", err
			}
			modulePath := modfile.ModulePath(dat)
			return filepath.ToSlash(filepath.Join(modulePath, strings.TrimPrefix(srcDir, currentDir))), nil
		}
	}
	// fall back to GOPATH mode
	goPaths := os.Getenv("GOPATH")
	if goPaths == "" {
		return "", fmt.Errorf("GOPATH is not set")
	}
	goPathList := strings.Split(goPaths, string(os.PathListSeparator))
	for _, goPath := range goPathList {
		sourceRoot := filepath.Join(goPath, "src") + string(os.PathSeparator)
		if strings.HasPrefix(srcDir, sourceRoot) {
			return filepath.ToSlash(strings.TrimPrefix(srcDir, sourceRoot)), nil
		}
	}
	return "", errOutsideGoPath
}
