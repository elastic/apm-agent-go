// +build go1.9

package apmgocql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuerySignature(t *testing.T) {
	assertSignatureEqual := func(expect, stmt string) {
		out := querySignature(stmt)
		assert.Equal(t, expect, out, "%s", stmt)
	}

	assertSignatureEqual("", "")
	assertSignatureEqual("", " ")

	assertSignatureEqual("DELETE FROM foo", "DELETE FROM foo USING TIMESTAMP")
	assertSignatureEqual("DELETE FROM foo.bar", "DELETE FROM foo.bar USING TIMESTAMP")
	assertSignatureEqual("DELETE FROM foo.bar", "DELETE column_name(term) FROM foo.bar USING TIMESTAMP")
	assertSignatureEqual("INSERT INTO foo", "INSERT INTO foo(x) VALUES(y)")
	assertSignatureEqual("INSERT INTO foo.bar", "INSERT INTO foo.bar(x,y) VALUES(y) IF NOT EXISTS")
	assertSignatureEqual("SELECT FROM foo", "SELECT * FROM foo")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM foo.bar WHERE baz")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT dateOf(created_at) AS creation_date FROM foo.bar WHERE baz")
	assertSignatureEqual("TRUNCATE foo.bar", "TRUNCATE foo.bar")
	assertSignatureEqual("TRUNCATE foo.bar", "TRUNCATE TABLE foo.bar")
	assertSignatureEqual("UPDATE foo", "UPDATE foo USING TIMESTAMP 123 SET bar=baz")
	assertSignatureEqual("UPDATE foo.bar", "UPDATE foo.bar SET baz=qux")

	assertSignatureEqual("DELETE", "DELETE :(")
	assertSignatureEqual("DELETE", "DELETE FROM :(")
	assertSignatureEqual("INSERT", "INSERT :(")
	assertSignatureEqual("INSERT", "INSERT INTO :(")
	assertSignatureEqual("SELECT", "SELECT (FROM) FROM :(")
	assertSignatureEqual("TRUNCATE", "TRUNCATE :(")
	assertSignatureEqual("UPDATE", "UPDATE :(")
	assertSignatureEqual("WHATEVER", "WHATEVER AND EVER")
}
