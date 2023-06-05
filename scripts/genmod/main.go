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
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.elastic.co/apm/v2"
)

var (
	versionFlag   = flag.String("version", "v"+apm.AgentVersion, "module version (e.g. \"v1.0.0\"")
	goVersionFlag = flag.String("go", "", "go version to expect in go.mod files")
	excludedPaths = flag.String("exclude", "tools", "paths to exclude. Separated by ,")
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
	if resolved, err := os.Readlink(root); err == nil {
		root = resolved
	}

	paths := strings.Split(*excludedPaths, ",")

	modules := make(map[string]*GoMod) // by module path
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		for _, p := range paths {
			dir := strings.TrimPrefix(path, root+"/")
			if dir == p {
				fmt.Fprintf(os.Stderr, "skipping %s\n", dir)
				return filepath.SkipDir
			}
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
		if name := info.Name(); name != root && strings.HasPrefix(name, ".") {
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
		fmt.Fprintf(os.Stderr, "# updating %s\n", gomod.Module.Path)
		if err := updateModule(absdir, gomod, modules); err != nil {
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
		relDir, err := filepath.Rel(dir, requireMod.dir)
		if err != nil {
			return err
		}
		args := []string{
			"mod", "edit",
			"-require", require.Path + "@" + *versionFlag,
			"-replace", require.Path + "=" + relDir,
		}
		cmd := exec.Command("go", args...)
		cmd.Stderr = os.Stderr
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("'go mod edit' failed: %w", err)
		}
	}
	cmd := exec.Command("go", "mod", "tidy", "-v", "-go", *goVersionFlag)
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("'go mod tidy' failed: %w", err)
	}
	return nil
}

// toposort topologically sorts the required modules, starting
// with the module specified by path.
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
	Go      string
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
