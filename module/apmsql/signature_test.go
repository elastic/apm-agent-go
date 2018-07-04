package apmsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuerySignature(t *testing.T) {
	assertSignatureEqual := func(expect, stmt string) {
		out := genericQuerySignature(stmt)
		assert.Equal(t, expect, out, "%s", stmt)
	}

	assertSignatureEqual("", "")
	assertSignatureEqual("", " ")

	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM foo.bar")
	assertSignatureEqual("SELECT FROM foo.bar.baz", "SELECT * FROM foo.bar.baz")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM `foo.bar`")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM \"foo.bar\"")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM [foo.bar]")
	assertSignatureEqual("SELECT FROM foo", "SELECT (x, y) FROM foo,bar,baz")
	assertSignatureEqual("SELECT FROM foo", "SELECT * FROM foo JOIN bar")
	assertSignatureEqual("SELECT FROM dollar$bill", "SELECT * FROM dollar$bill")
	assertSignatureEqual("SELECT FROM myta\n-æøåble", "SELECT id FROM \"myta\n-æøåble\" WHERE id = 2323")
	assertSignatureEqual("SELECT FROM foo.bar", "SELECT * FROM foo-- abc\n./*def*/bar")

	// We capture the first table of the outermost select statement.
	assertSignatureEqual("SELECT FROM table1", "SELECT *,(SELECT COUNT(*) FROM table2 WHERE table2.field1 = table1.id) AS count FROM table1 WHERE table1.field1 = 'value'")

	// If the outermost select operates on derived tables, then we
	// just return "SELECT" (i.e. the fallback).
	assertSignatureEqual("SELECT", "SELECT * FROM (SELECT foo FROM bar) AS foo_bar")

	assertSignatureEqual("DELETE FROM foo.bar", "DELETE FROM foo.bar WHERE baz=1")
	assertSignatureEqual("UPDATE foo.bar", "UPDATE IGNORE foo.bar SET bar=1 WHERE baz=2")
	assertSignatureEqual("INSERT INTO foo.bar", "INSERT INTO foo.bar (col) VALUES(?)")
	assertSignatureEqual("INSERT INTO foo.bar", "INSERT LOW_PRIORITY IGNORE INTO foo.bar (col) VALUES(?)")
	assertSignatureEqual("CALL foo", "CALL foo(bar, 123)")

	// For all of the below (DDL, miscellaneous, and broken statements)
	// we just capture the initial token:
	assertSignatureEqual("ALTER", "ALTER TABLE foo ADD ()")
	assertSignatureEqual("CREATE", "CREATE TABLE foo ...")
	assertSignatureEqual("DROP", "DROP TABLE foo")
	assertSignatureEqual("SAVEPOINT", "SAVEPOINT x_asd1234")
	assertSignatureEqual("BEGIN", "BEGIN")
	assertSignatureEqual("COMMIT", "COMMIT")
	assertSignatureEqual("ROLLBACK", "ROLLBACK")
	assertSignatureEqual("SELECT", "SELECT * FROM (SELECT EOF")
	assertSignatureEqual("SELECT", "SELECT 'neverending literal FROM (SELECT * FROM ...")
}

func BenchmarkQuerySignature(b *testing.B) {
	sql := "SELECT *,(SELECT COUNT(*) FROM table2 WHERE table2.field1 = table1.id) AS count FROM table1 WHERE table1.field1 = 'value'"
	for i := 0; i < b.N; i++ {
		signature := genericQuerySignature(sql)
		if signature != "SELECT FROM table1" {
			panic("unexpected result: " + signature)
		}
		b.SetBytes(int64(len(sql)))
	}
}
