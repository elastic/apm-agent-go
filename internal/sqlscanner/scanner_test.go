package sqlscanner

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	Input   string `json:"input"`
	Tokens  []struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	} `json:"tokens,omitempty"`
}

func TestScanner(t *testing.T) {
	var tests []test
	data, err := ioutil.ReadFile("testdata/tests.json")
	require.NoError(t, err)
	err = json.Unmarshal(data, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			msgFormat := "%s"
			args := []interface{}{test.Input}
			if test.Comment != "" {
				msgFormat += " (%s)"
				args = append(args, test.Comment)
			}

			s := NewScanner(test.Input)
			for _, tok := range test.Tokens {
				if !assert.True(t, s.Scan()) {
					return
				}
				assert.Equal(t, tok.Kind, s.Token().String())
				assert.Equal(t, tok.Text, s.Text())
			}
			assert.False(t, s.Scan())
		})
	}
}
