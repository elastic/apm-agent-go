package apm_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
	"go.elastic.co/fastjson"
)

var (
	flagDumpMetadata = flag.Bool("dump-metadata", false, "Dump metadata and exit without running any tests")
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	flag.Parse()
	if *flagDumpMetadata {
		dumpMetadata()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func getSubprocessMetadata(t *testing.T, env ...string) (*model.System, *model.Process, *model.Service) {
	cmd := exec.Command(os.Args[0], "-dump-metadata")
	cmd.Env = append(os.Environ(), env...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if !assert.NoError(t, cmd.Run()) {
		t.FailNow()
	}

	var system model.System
	var process model.Process
	var service model.Service

	output := stdout.String()
	d := json.NewDecoder(&stdout)
	if !assert.NoError(t, d.Decode(&system)) {
		t.Logf("output: %q", output)
		t.FailNow()
	}
	require.NoError(t, d.Decode(&process))
	require.NoError(t, d.Decode(&service))
	return &system, &process, &service
}

func dumpMetadata() {
	var transport transporttest.RecorderTransport
	tracer, _ := apm.NewTracer("", "")
	defer tracer.Close()
	tracer.Transport = &transport

	tracer.StartTransaction("name", "type").End()
	tracer.Flush(nil)
	system, process, service := transport.Metadata()

	var w fastjson.Writer
	for _, m := range []fastjson.Marshaler{&system, &process, &service} {
		if err := m.MarshalFastJSON(&w); err != nil {
			panic(err)
		}
	}
	os.Stdout.Write(w.Bytes())
}
