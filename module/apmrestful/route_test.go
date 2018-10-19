package apmrestful

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMassageRoutePath(t *testing.T) {
	out := massageRoutePath("/articles/{category}/{id:[0-9]+}")
	assert.Equal(t, "/articles/{category}/{id}", out)
}
