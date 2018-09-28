package elasticapm

import (
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/apm-agent-go/internal/apmstrings"
	"github.com/elastic/apm-agent-go/model"
)

var (
	currentProcess model.Process
	goAgent        = model.Agent{Name: "go", Version: AgentVersion}
	goLanguage     = model.Language{Name: "go", Version: runtime.Version()}
	goRuntime      = model.Runtime{Name: runtime.Compiler, Version: runtime.Version()}
	localSystem    model.System

	serviceNameInvalidRegexp = regexp.MustCompile("[^" + serviceNameValidClass + "]")
)

const (
	envHostname = "ELASTIC_APM_HOSTNAME"

	serviceNameValidClass = "a-zA-Z0-9 _-"
)

func init() {
	currentProcess = getCurrentProcess()
	localSystem = getLocalSystem()
}

func getCurrentProcess() model.Process {
	ppid := os.Getppid()
	title, err := currentProcessTitle()
	if err != nil {
		title = os.Args[0]
	}
	return model.Process{
		Pid:   os.Getpid(),
		Ppid:  &ppid,
		Title: truncateKeyword(title),
		Argv:  os.Args,
	}
}

func makeService(name, version, environment string) model.Service {
	return model.Service{
		Name:        truncateKeyword(name),
		Version:     truncateKeyword(version),
		Environment: truncateKeyword(environment),
		Agent:       goAgent,
		Language:    &goLanguage,
		Runtime:     &goRuntime,
	}
}

func getLocalSystem() model.System {
	system := model.System{
		Architecture: runtime.GOARCH,
		Platform:     runtime.GOOS,
	}
	system.Hostname = os.Getenv(envHostname)
	if system.Hostname == "" {
		if hostname, err := os.Hostname(); err == nil {
			system.Hostname = hostname
		}
	}
	system.Hostname = truncateKeyword(system.Hostname)
	return system
}

func validTagKey(k string) bool {
	return !strings.ContainsAny(k, `.*"`)
}

func validateServiceName(name string) error {
	idx := serviceNameInvalidRegexp.FindStringIndex(name)
	if idx == nil {
		return nil
	}
	return errors.Errorf(
		"invalid service name %q: character %q is not in the allowed set (%s)",
		name, name[idx[0]], serviceNameValidClass,
	)
}

func sanitizeServiceName(name string) string {
	return serviceNameInvalidRegexp.ReplaceAllString(name, "_")
}

func truncateKeyword(s string) string {
	// At the time of writing, all keyword length
	// limits are 1024, enforced by JSON Schema.
	return apmstrings.Truncate(s, 1024)
}

func truncateText(s string) string {
	// Non-keyword string fields should be limited
	// to 10000 characters/runes. This is not
	// currently enforced by JSON Schema, as we
	// may want to make it configurable.
	return apmstrings.Truncate(s, 10000)
}

func nextGracePeriod(p time.Duration) time.Duration {
	if p == -1 {
		return 0
	}
	for i := time.Duration(0); i < 6; i++ {
		if p == (i * i * time.Second) {
			return (i + 1) * (i + 1) * time.Second
		}
	}
	return p
}

// jitterDuration returns d +/- some multiple of d in the range [0,j].
func jitterDuration(d time.Duration, rng *rand.Rand, j float64) time.Duration {
	if d == 0 || j == 0 {
		return d
	}
	r := (rng.Float64() * j * 2) - j
	return d + time.Duration(float64(d)*r)
}
