package factory

import (
	"context"
	"testing"
)

func TestDatabase(t *testing.T) {
	assert := func(result bool, format string, params ...interface{}) {
		if !result {
			t.Errorf(format, params...)
		}
	}

	db := DB{}
	assert(db.DB == nil, "DB instance expected nil")
	assert(db.Profiler == nil, "DB profiler expected nil")

	db.Profiler = &DatabaseProfilerStdout{}

	ctxdb := db.With(context.WithValue(context.Background(), "foo", "bar"))
	if ctxvalue := ctxdb.ctx.Value("foo"); ctxvalue == nil {
		t.Errorf("Expected context with value foo")
	} else {
		if ctxvalue.(string) != "bar" {
			t.Errorf("Expected context with value foo=bar")
		}
	}

	assert(db.Quiet().Profiler == nil, "DB quiet profiler expected nil")

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
			assert(ok, "Expected existance of set[%s]", key)
			if ok {
				assert(val == ":"+key, "Expected value of set[%s] = :%s, got %s", key, key, val)
			}
		}
		check("id")
		check("name")
	}

	// check set functionality
	{
		i := db.set(&dbStruct)
		assert(i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result: %s", i)
	}
	{
		i := db.set(&dbStruct, "id")
		assert(i == "id=:id", "Unexpected set() result with allowed=[id]: %s", i)
	}
	{
		i := db.set(&dbStruct, "id", "name")
		assert(i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result with allowed=[id,name]: %s", i)
	}
	{
		i := db.set(&dbStruct)
		assert(i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected set() result: %s", i)
	}
	{
		i := db.setImplode(", ", set)
		assert(i == "id=:id, name=:name" || i == "name=:name, id=:id", "Unexpected setImplode() 1 result: %s", i)
	}
	{
		i := db.setImplode(" AND ", set)
		assert(i == "id=:id AND name=:name" || i == "name=:name AND id=:id", "Unexpected setImplode() 2 result: %s", i)
	}
	{
		delete(set, "name")
		i := db.setImplode(" AND ", set)
		assert(i == "id=:id", "Unexpected setImplode() 3 result: %s", i)
	}
}
