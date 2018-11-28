package sqlutil_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/internal/sqlutil"
)

type test struct {
	Comment string `json:"comment,omitempty"`
	Input   string `json:"input"`
	Output  string `json:"output"`
}

func TestQuerySignature(t *testing.T) {
	var tests []test
	data, err := ioutil.ReadFile("testdata/tests.json")
	require.NoError(t, err)
	err = json.Unmarshal(data, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		msgFormat := "%s"
		args := []interface{}{test.Input}
		if test.Comment != "" {
			msgFormat += " (%s)"
			args = append(args, test.Comment)
		}
		out := sqlutil.QuerySignature(test.Input)
		if assert.Equal(t, test.Output, out, append([]interface{}{msgFormat}, args...)) {
			if test.Comment != "" {
				t.Logf("// %s", test.Comment)
			}
			t.Logf("%q => %q", test.Input, test.Output)
		}
	}
}

func BenchmarkQuerySignature(b *testing.B) {
	sql := "SELECT *,(SELECT COUNT(*) FROM table2 WHERE table2.field1 = table1.id) AS count FROM table1 WHERE table1.field1 = 'value'"
	for i := 0; i < b.N; i++ {
		signature := sqlutil.QuerySignature(sql)
		if signature != "SELECT FROM table1" {
			panic("unexpected result: " + signature)
		}
		b.SetBytes(int64(len(sql)))
	}
}
