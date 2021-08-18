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

//go:build go1.9
// +build go1.9

package apmgodog_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/cucumber/godog/gherkin"
)

// Run runs the Gherkin feature files specified in paths as Go subtests.
func Run(t *testing.T, paths []string) {
	initContext := func(s *godog.Suite) {
		var commands chan command
		var scenarioFailed bool
		s.BeforeFeature(func(f *gherkin.Feature) {
			commands = make(chan command)
			go runCommands(t, commands)
			startTest(commands, f.Name)
		})
		s.AfterFeature(func(f *gherkin.Feature) {
			endTest(commands, nil)
			close(commands)
		})
		s.BeforeScenario(func(s interface{}) {
			scenarioFailed = false
			switch s := s.(type) {
			case *gherkin.Scenario:
				startTest(commands, s.Name)
			case *gherkin.ScenarioOutline:
				startTest(commands, s.Name)
			}
		})
		s.AfterScenario(func(_ interface{}, err error) {
			endTest(commands, err)
		})
		s.BeforeStep(func(step *gherkin.Step) {
			if scenarioFailed {
				fmt.Printf(colors.Yellow("    %s%s\n"), step.Keyword, step.Text)
			}
		})
		s.AfterStep(func(step *gherkin.Step, err error) {
			if err != nil {
				scenarioFailed = true
				fmt.Printf(colors.Red("    %s%s (%s)\n"), step.Keyword, step.Text, err)
			} else {
				fmt.Printf(colors.Cyan("    %s%s\n"), step.Keyword, step.Text)
			}
		})
		InitContext(s)
	}

	godog.RunWithOptions("godog", initContext, godog.Options{
		Format: "events", // must pick one, this will do
		Paths:  paths,
		Output: ioutil.Discard,
	})
}

func startTest(commands chan command, name string) {
	commands <- func(t *testing.T) error {
		t.Run(name, func(t *testing.T) {
			runCommands(t, commands)
		})
		return nil
	}
}

func endTest(commands chan command, err error) {
	commands <- func(t *testing.T) error {
		if err != nil {
			return err
		}
		return done{}
	}
}

func runCommands(t *testing.T, commands chan command) {
	for {
		cmd, ok := <-commands
		if !ok {
			return
		}
		err := cmd(t)
		switch err.(type) {
		case nil:
		case done:
			return
		default:
			t.Fatal(err)
		}
	}
}

type command func(t *testing.T) error

type done struct{}

func (done) Error() string { return "done" }
