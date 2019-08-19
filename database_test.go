// +build integration

package factory

import (
	"context"
	"log"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/titpetric/factory/logger"
)

func assert(t *testing.T, result bool, format string, params ...interface{}) {
	if !result {
		t.Errorf(format, params...)
	}
}

func TestTransactions(t *testing.T) {
	var err error

	ctx1 := context.WithValue(context.Background(), "testing", true)
	ctx2 := context.Background()

	db1 := Database.MustGet().With(ctx1)
	db1.SetLogger(logger.Default{})
	db2 := Database.MustGet().With(ctx2)
	db2.SetLogger(logger.Default{})

	// set up db tables
	_, err = db1.Exec("create table innodb_deadlock_maker_a(a int primary key, b varchar(255)) engine=innodb;")
	assert(t, err == nil, "error: %+v", err)
	_, err = db1.Exec("insert into innodb_deadlock_maker_a (a, b) values (1, ''), (2, '');")
	assert(t, err == nil, "error: %+v", err)

	done := make(chan bool)

	// first deadlock window
	go func() {
		err = db1.Transaction(func() error {
			if _, err := db1.Exec("/* db1 */ select * from innodb_deadlock_maker_a where a=1 for share"); err != nil {
				return err
			}
			if _, err := db1.Exec("/* db1 */ select sleep(0.2)"); err != nil {
				return err
			}
			if _, err := db1.Exec("/* db1 */ delete from innodb_deadlock_maker_a where a=1"); err != nil {
				return err
			}
			return nil
		})
		assert(t, err == nil, "db1 error: %+v", err)
		log.Println("Finished db1")
		done <- true
	}()

	// second deadlock window
	go func() {
		err = db2.Transaction(func() error {
			if _, err := db2.Exec("/* db2 */ select sleep(0.1)"); err != nil {
				return err
			}
			if _, err := db2.Exec("/* db2 */ delete from innodb_deadlock_maker_a where a=1"); err != nil {
				return err
			}
			return nil
		})
		assert(t, err == nil, "db2 error: %+v", err)
		log.Println("Finished db2")
		done <- true
	}()

	<-done
	<-done
}

func TestDatabase(t *testing.T) {
	db := &DB{}
	assert(t, db.DB == nil, "DB instance expected nil")

	// check that db conforms to execer interface
	var _ sqlx.Execer = db

	ctxdb := db.With(context.WithValue(context.Background(), "foo", "bar"))
	if ctxvalue := ctxdb.ctx.Value("foo"); ctxvalue == nil {
		t.Errorf("Expected context with value foo")
	} else {
		if ctxvalue.(string) != "bar" {
			t.Errorf("Expected context with value foo=bar")
		}
	}

	dbStruct := struct {
		ID    int    `db:"id"`
		Name  string `db:"name,omitempty"`
		Title string `db:"-"`
	}{123, "Tit Petric", "Sir"}

	set := db.setMap(&dbStruct)

	if len(set) != 2 {
		t.Errorf("Expected setmap length 2, got %d", len(set))
	}

	// check expected contents of set
	{
		check := func(key string) {
			val, ok := set[key]
			assert(t, ok, "Expected existance of set[%s]", key)
			if ok {
				assert(t, val == ":"+key, "Expected value of set[%s] = :%s, got %s", key, key, val)
			}
		}
		check("id")
		check("name")
	}

	// check set functionality
	{
		i := db.set(&dbStruct)
		assert(t, i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result: %s", i)
	}
	{
		i := db.set(&dbStruct, "id")
		assert(t, i == "id=:id", "Unexpected set() result with allowed=[id]: %s", i)
	}
	{
		i := db.set(&dbStruct, "id", "name")
		assert(t, i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result with allowed=[id,name]: %s", i)
	}
	{
		i := db.set(&dbStruct)
		assert(t, i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result: %s", i)
	}
	{
		i := db.setImplode(", ", set)
		assert(t, i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected setImplode() 1 result: %s", i)
	}
	{
		i := db.setImplode(" AND ", set)
		assert(t, i == "id=:id AND name=:name" || i == "name=:name AND id=:id", "Unexpected setImplode() 2 result: %s", i)
	}
	{
		delete(set, "name")
		i := db.setImplode(" AND ", set)
		assert(t, i == "id=:id", "Unexpected setImplode() 3 result: %s", i)
	}
}
