package gluamapper

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yuin/gopher-lua"
)

func errorIfNotEqual(t *testing.T, v1, v2 any) {
	if v1 != v2 {
		_, file, line, _ := runtime.Caller(1)
		t.Errorf("%v line %v: '%v' expected, but got '%v'", filepath.Base(file), line, v1, v2)
	}
}

type testRole struct {
	Name string
}

type testPerson struct {
	Name      string
	Age       int
	WorkPlace string
	Role      []*testRole
}

type testStruct struct {
	Nil    any
	Bool   bool
	String string
	Number int `gluamapper:"number_value"`
	Func   any
}

func TestMap(t *testing.T) {
	t.Parallel()

	L := lua.NewState()
	if err := L.DoString(`
    person = {
      name = "Michel",
      age  = "31", -- weakly input
      work_place = "San Jose",
      role = {
        {
          name = "Administrator"
        },
        {
          name = "Operator"
        }
      }
    }
	`); err != nil {
		t.Error(err)
	}
	var person testPerson
	if err := Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, "Michel", person.Name)
	errorIfNotEqual(t, 31, person.Age)
	errorIfNotEqual(t, "San Jose", person.WorkPlace)
	errorIfNotEqual(t, 2, len(person.Role))
	errorIfNotEqual(t, "Administrator", person.Role[0].Name)
	errorIfNotEqual(t, "Operator", person.Role[1].Name)
}

func TestTypes(t *testing.T) {
	t.Parallel()

	L := lua.NewState()
	if err := L.DoString(`
    tbl = {
      ["Nil"] = nil,
      ["Bool"] = true,
      ["String"] = "string",
      ["number_value"] = 10,
      ["Func"] = function() end
    }
	`); err != nil {
		t.Error(err)
	}
	var stct testStruct

	if err := NewMapper(Option{NameFunc: Identity}).Map(L.GetGlobal("tbl").(*lua.LTable), &stct); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, nil, stct.Nil)
	errorIfNotEqual(t, true, stct.Bool)
	errorIfNotEqual(t, "string", stct.String)
	errorIfNotEqual(t, 10, stct.Number)
}

func TestNameFunc(t *testing.T) {
	t.Parallel()

	L := lua.NewState()
	if err := L.DoString(`
    person = {
      Name = "Michel",
      Age  = "31", -- weekly input
      WorkPlace = "San Jose",
      Role = {
        {
          Name = "Administrator"
        },
        {
          Name = "Operator"
        }
      }
    }
	`); err != nil {
		t.Error(err)
	}
	var person testPerson
	mapper := NewMapper(Option{NameFunc: Identity})
	if err := mapper.Map(L.GetGlobal("person").(*lua.LTable), &person); err != nil {
		t.Error(err)
	}
	errorIfNotEqual(t, "Michel", person.Name)
	errorIfNotEqual(t, 31, person.Age)
	errorIfNotEqual(t, "San Jose", person.WorkPlace)
	errorIfNotEqual(t, 2, len(person.Role))
	errorIfNotEqual(t, "Administrator", person.Role[0].Name)
	errorIfNotEqual(t, "Operator", person.Role[1].Name)
}

func TestError(t *testing.T) {
	t.Parallel()

	L := lua.NewState()
	tbl := L.NewTable()
	L.SetField(tbl, "key", lua.LString("value"))
	err := Map(tbl, 1)
	if err.Error() != "result must be a pointer" {
		t.Error("invalid error message")
	}

	tbl = L.NewTable()
	tbl.Append(lua.LNumber(1))
	var person testPerson
	err = Map(tbl, &person)
	if err.Error() != "arguments #1 must be a table, but got an array" {
		t.Error("invalid error message")
	}
}

func TestFromGoValue(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		value any
		test  string
	}{
		"nil": {
			value: nil,
			test:  `return function(v) assert(v == nil, "v is not nil") end`,
		},
		"true": {
			value: true,
			test:  `return function(v) assert(v == true, "v is not true") end`,
		},
		"false": {
			value: false,
			test:  `return function(v) assert(v == false, "v is not false") end`,
		},
		"string": {
			value: "foo",
			test:  `return function(v) assert(v == "foo", "v is not 'foo'") end`,
		},
		"int": {
			value: int(69),
			test:  `return function(v) assert(v == 69, "v is not 69") end`,
		},
		"int64": {
			value: int64(69),
			test:  `return function(v) assert(v == 69, "v is not 69") end`,
		},
		"uint": {
			value: uint(69),
			test:  `return function(v) assert(v == 69, "v is not 69") end`,
		},
		"uint64": {
			value: uint64(69),
			test:  `return function(v) assert(v == 69, "v is not 69") end`,
		},
		"float64": {
			value: float64(69),
			test:  `return function(v) assert(v == 69, "v is not 69") end`,
		},
		"array": {
			value: [2]string{"foo", "bar"},
			test: `return function(v)
				assert(#v == 2, "v does not have length 2")
				assert(v[1] == "foo", "v[1] is not 'foo'")
				assert(v[2] == "bar", "v[2] is not 'bar'")
			end`,
		},
		"slice": {
			value: []string{"foo", "bar"},
			test: `return function(v)
				assert(#v == 2, "v does not have length 2")
				assert(v[1] == "foo", "v[1] is not 'foo'")
				assert(v[2] == "bar", "v[2] is not 'bar'")
			end`,
		},
		"map": {
			value: map[string]int{
				"foo": 69,
				"bar": 420,
			},
			test: `return function(v)
				assert(v.foo == 69, "v.foo is not 69")
				assert(v.bar == 420, "v.bar is not 420")
			end`,
		},
		"struct": {
			value: struct {
				Foo string
				Bar int
			}{
				Foo: "baz",
				Bar: 69,
			},
			test: `return function(v)
				assert(v.Foo == "baz", "v.Foo is not 'baz'")
				assert(v.Bar == 69, "v.Bar is not 69")
			end`,
		},
		"struct tags": {
			value: struct {
				Foo     string `gluamapper:"foo"`
				Bar     int    `gluamapper:"bar"`
				Omitted bool   `gluamapper:"-"`
			}{
				Foo:     "baz",
				Bar:     69,
				Omitted: true,
			},
			test: `return function(v)
				assert(v.foo == "baz", "v.foo is not 'baz'")
				assert(v.bar == 69, "v.bar is not 69")
				assert(v.Omitted == nil, "v.Omitted was not omitted")
				assert(v.omitted == nil, "v.omitted was not omitted")
			end`,
		},
		"struct pointer": {
			value: &struct {
				Foo string
				Bar int
			}{
				Foo: "baz",
				Bar: 69,
			},
			test: `return function(v)
				assert(v.Foo == "baz", "v.Foo is not 'baz'")
				assert(v.Bar == 69, "v.Bar is not 69")
			end`,
		},
		"struct interface": {
			value: any(struct {
				Foo string
				Bar int
			}{
				Foo: "baz",
				Bar: 69,
			}),
			test: `return function(v)
				assert(v.Foo == "baz", "v.Foo is not 'baz'")
				assert(v.Bar == 69, "v.Bar is not 69")
			end`,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			L := lua.NewState()

			lv := FromGoValue(L, tc.value)

			err := L.DoString(tc.test)
			require.NoError(t, err)

			testFunc := L.Get(-1).(*lua.LFunction)
			co, _ := L.NewThread()
			st, err, _ := L.Resume(co, testFunc, lv)
			require.NoError(t, err)
			require.Equal(t, lua.ResumeOK, st)
		})
	}
}
