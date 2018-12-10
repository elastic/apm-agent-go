package apmbeego_test

import (
	"context"
	"testing"

	"github.com/astaxie/beego/orm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.elastic.co/apm/apmtest"
	"go.elastic.co/apm/module/apmsql"
	_ "go.elastic.co/apm/module/apmsql/sqlite3"
)

type User struct {
	Id   int    `orm:"auto"`
	Name string `orm:"size(100)"`
}

func init() {
	orm.RegisterDriver(apmsql.DriverPrefix+"sqlite3", orm.DRSqlite)
	orm.RegisterDataBase("default", apmsql.DriverPrefix+"sqlite3", ":memory:", 30)
	orm.RegisterModel(&User{})
}

func TestORM(t *testing.T) {
	err := orm.RunSyncdb("default", false, false)
	require.NoError(t, err)

	o := orm.NewOrm()
	_, spans, _ := apmtest.WithTransaction(func(ctx context.Context) {
		_, err := o.Insert(&User{Name: "birgit"})
		assert.NoError(t, err)
	})

	// Sadly, there is no way to propagate context to the underlying
	// database/sql queries executed by beego/orm. We should at least
	// provide a way of instrumenting beego/orm.Ormer.
	require.Len(t, spans, 0)
}
