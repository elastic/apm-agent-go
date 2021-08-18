// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"go.elastic.co/apm"
)

var (
	checkFlag   = flag.Bool("check", false, "check the go.mod files are complete, instead of updating them")
	versionFlag = flag.String("version", "v"+apm.AgentVersion, "module version (e.g. \"v1.0.0\"")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <dir>\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	// Locate and parse all go.mod files.
	root, err := filepath.Abs(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	modules := make(map[string]*GoMod) // by module path
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if info.Name() == "go.mod" {
				gomod, err := readGoMod(path)
				if err != nil {
					return err
				}
				modules[gomod.Module.Path] = gomod
			}
			return nil
		}
		name := info.Name()
		if name != root && (name == "vendor" || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	seen := make(map[string]bool)
	var modulePaths []string
	for path := range modules {
		toposort(path, modules, seen, &modulePaths)
	}
	for _, modpath := range modulePaths {
		gomod := modules[modpath]
		absdir, err := filepath.Abs(gomod.dir)
		if err != nil {
			log.Fatal(err)
		}
		if !*checkFlag {
			fmt.Fprintf(os.Stderr, "# updating %s\n", gomod.Module.Path)
			if err := updateModule(absdir, gomod, modules); err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "# checking %s\n", gomod.Module.Path)
			if err := checkModule(absdir, gomod, modules); err != nil {
				log.Fatal(err)
			}
		}
		if err := checkModuleComplete(absdir, gomod, modules); err != nil {
			log.Fatal(err)
		}
	}
}

func updateModule(dir string, gomod *GoMod, modules map[string]*GoMod) error {
	for _, require := range gomod.Require {
		requireMod, ok := modules[require.Path]
		if !ok {
			continue
		}
		if require.Version == *versionFlag {
			continue
		}
		relDir, err := filepath.Rel(dir, requireMod.dir)
		if err != nil {
			return err
		}
		cmd := exec.Command(
			"go", "mod", "edit",
			"-require", require.Path+"@"+*versionFlag,
			"-replace", require.Path+"="+relDir,
		)
		cmd.Env = append(os.Environ(), "GO111MODULE=on")
		cmd.Env = append(cmd.Env, "GOPROXY=http://proxy.invalid", "GOSUMDB=sum.golang.org https://sum.golang.org")
		cmd.Stderr = os.Stderr
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

// checkModule checks that the require stanzas in $dir/go.mod have the
// correct versions, and have appropriate matching "replace" stanzas.
func checkModule(dir string, gomod *GoMod, modules map[string]*GoMod) error {
	// Verify that any required module in modules has the version
	// specified in versionFlag, and has a replacement stanza.
	var gomodBad bool
	for _, require := range gomod.Require {
		requireMod, ok := modules[require.Path]
		if !ok {
			continue
		}
		if require.Version != *versionFlag {
			fmt.Fprintf(
				os.Stderr,
				" - found \"require %s %s\", expected %s\n",
				require.Path, require.Version, *versionFlag,
			)
			gomodBad = true
		}
		relDir, err := filepath.Rel(dir, requireMod.dir)
		if err != nil {
			return err
		}
		var foundReplace bool
		for _, replace := range gomod.Replace {
			if replace.Old.Path == require.Path && replace.Old.Version == "" {
				if filepath.Clean(replace.New.Path) != relDir {
					fmt.Fprintf(
						os.Stderr,
						" - found \"replace %s => %s\", expected %s\n",
						replace.Old.Path, replace.New.Path, relDir,
					)
					gomodBad = true
				}
				foundReplace = true
				break
			}
		}
		if !foundReplace {
			fmt.Fprintf(os.Stderr, " - missing \"replace %s => %s\"\n", require.Path, relDir)
			gomodBad = true
		}
	}
	if gomodBad {
		return errors.Errorf("%s/go.mod invalid", gomod.dir)
	}
	return nil
}

// checkModuleComplete checks that $dir/go.mod is complete by running
// "go build" and "go mod tidy", ensuring no changes are required.
func checkModuleComplete(dir string, gomod *GoMod, modules map[string]*GoMod) error {
	// Make sure we have all of the module's dependencies first.
	cmd := exec.Command("go", "mod", "download")
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stderr = os.Stderr
	cmd.Dir = gomod.dir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "'go mod download' failed")
	}

	// Check we can build the module's tests and its transitive dependencies
	// without updating go.mod.
	cmd = exec.Command("go", "test", "-c", "-mod=readonly", "-o", os.DevNull)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Env = append(cmd.Env, "GOPROXY=http://proxy.invalid", "GOSUMDB=sum.golang.org https://sum.golang.org")
	cmd.Stderr = os.Stderr
	cmd.Dir = gomod.dir
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "'go test' failed")
	}

	// We create a temporary program which imports the module, and then
	// use "go mod tidy", checking if go.mod is changed. "go mod tidy"
	// can require more packages than the previous "go build".
	tmpdir, err := ioutil.TempDir("", "genmod")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	tmpGomodPath := filepath.Join(tmpdir, "go.mod")
	tmpGomainPath := filepath.Join(tmpdir, "main.go")
	tmpGomainContent := []byte(fmt.Sprintf(`
package main
import _ %q
func main() {}
`, gomod.Module.Path))

	var tmpGomodContent bytes.Buffer
	fmt.Fprintln(&tmpGomodContent, "module main")
	required := required(gomod.Module.Path, modules)
	sort.Strings(required)
	for _, path := range required {
		if path == gomod.Module.Path {
			fmt.Fprintf(&tmpGomodContent, "\nrequire %s %s", path, *versionFlag)
			fmt.Fprintln(&tmpGomodContent)
		}
	}
	for _, path := range required {
		gomod := modules[path]
		fmt.Fprintf(&tmpGomodContent, "\nreplace %s => %s\n", path, gomod.dir)
	}
	// Add "go <version>", using the latest release tag.
	tags := build.Default.ReleaseTags
	// TODO(stn): go1.17 introduced changes to gomod, which breaks this
	// check. Lock go1.17 and higher to go1.16 behavior for now.
	tag := tags[len(tags)-1][2:]
	minorVersion, err := strconv.Atoi(tag[2:4])
	if err != nil {
		return err
	}
	if tag == "1.17" {
		tag = "1.16"
	}
	fmt.Fprintf(&tmpGomodContent, "\ngo %s\n", tag)

	if err := ioutil.WriteFile(tmpGomodPath, tmpGomodContent.Bytes(), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile(tmpGomainPath, tmpGomainContent, 0644); err != nil {
		return err
	}

	cmd = exec.Command("go", "mod", "tidy", "-v")
	if minorVersion > 16 {
		cmd.Args = append(cmd.Args, "-go=1.16")
	}
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Env = append(cmd.Env, "GOPROXY=http://proxy.invalid", "GOSUMDB=sum.golang.org https://sum.golang.org")
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("diff", "-c", "-", "--label=old", tmpGomodPath, "--label=new")
	cmd.Stdin = bytes.NewReader(tmpGomodContent.Bytes())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// required returns the transitive modules dependencies of
// the module specified by path, including path itself.
func required(path string, modules map[string]*GoMod) []string {
	var paths []string
	toposort(path, modules, make(map[string]bool), &paths)
	return paths
}

// toposort topologically sorts the required modules, starting
// with the moduled specified by path.
func toposort(path string, modules map[string]*GoMod, seen map[string]bool, paths *[]string) {
	if seen[path] {
		return
	}
	gomod := modules[path]
	if gomod == nil {
		return
	}
	seen[path] = true
	for _, require := range gomod.Require {
		toposort(require.Path, modules, seen, paths)
	}
	*paths = append(*paths, path)
}

func readGoMod(path string) (*GoMod, error) {
	cmd := exec.Command("go", "mod", "edit", "-json", path)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	gomod := GoMod{dir: filepath.Dir(path)}
	if err := json.Unmarshal(output, &gomod); err != nil {
		return nil, err
	}
	return &gomod, nil
}

// GoMod holds definition of a Go module, as in go.mod.
type GoMod struct {
	// dir is the directory containing go.mod
	dir string

	Module  Module
	Require []Require
	Exclude []Module
	Replace []Replace
}

// Require describes a "require" go.mod stanza.
type Require struct {
	Path     string
	Version  string
	Indirect bool
}

// Replace describes a "replace" go.mod stanza.
type Replace struct {
	Old Module
	New Module
}

// Module describes a Go module path and, optionally, version.
type Module struct {
	Path    string
	Version string
}
