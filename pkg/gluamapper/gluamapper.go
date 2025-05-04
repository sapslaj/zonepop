// gluamapper provides an easy way to map GopherLua tables to Go structs.
package gluamapper

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

const StructTagName = "gluamapper"

// Option is a configuration that is used to create a new mapper.
type Option struct {
	// Function to convert a lua table key to Go's one. This defaults to "Id".
	NameFunc func(string) string

	DecoderConfig *mapstructure.DecoderConfig
}

// Mapper maps a lua table to a Go struct pointer.
type Mapper struct {
	Option Option
}

// NewMapper returns a new mapper.
func NewMapper(opt Option) *Mapper {
	return &Mapper{opt}
}

// Map maps the lua table to the given struct pointer.
func (mapper *Mapper) Map(tbl *lua.LTable, st any) error {
	opt := mapper.Option
	mp, ok := ToGoValue(tbl, opt).(map[string]any)
	if !ok {
		return errors.New("arguments #1 must be a table, but got an array")
	}
	config := mapper.Option.DecoderConfig
	if config == nil {
		config = &mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
		}
	}
	config.TagName = StructTagName
	if config.MatchName == nil {
		config.MatchName = MatchSnakeCase
	}
	config.Result = st
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(mp)
}

// Map maps the lua table to the given struct pointer with default options.
func Map(tbl *lua.LTable, st any) error {
	return NewMapper(Option{}).Map(tbl, st)
}

// Identity is an Option.NameFunc that returns given string as-is.
func Identity(s string) string {
	return s
}

var camelre = regexp.MustCompile(`_([a-z])`)

// ToUpperCamelCase is an Option.NameFunc that converts strings from snake case to upper camel case.
func ToUpperCamelCase(s string) string {
	return strings.ToUpper(string(s[0])) + camelre.ReplaceAllStringFunc(
		s[1:],
		func(s string) string {
			return strings.ToUpper(s[1:])
		},
	)
}

func MatchSnakeCase(mapKey, fieldName string) bool {
	return strings.EqualFold(
		strings.ReplaceAll(mapKey, "_", ""),
		strings.ReplaceAll(fieldName, "_", ""),
	)
}

// ToGoValue converts the given LValue to a Go object.
func ToGoValue(lv lua.LValue, opt Option) any {
	nameFunc := opt.NameFunc
	if nameFunc == nil {
		nameFunc = Identity
	}
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn == 0 { // table
			ret := make(map[string]any)
			v.ForEach(func(key, value lua.LValue) {
				keystr := fmt.Sprint(ToGoValue(key, opt))
				ret[nameFunc(keystr)] = ToGoValue(value, opt)
			})
			return ret
		} else { // array
			ret := make([]any, 0, maxn)
			for i := 1; i <= maxn; i++ {
				ret = append(ret, ToGoValue(v.RawGetInt(i), opt))
			}
			return ret
		}
	default:
		return v
	}
}

func (mapper *Mapper) ToGoValue(lv lua.LValue) any {
	return ToGoValue(lv, mapper.Option)
}

func FromGoValue(L *lua.LState, v any) lua.LValue {
	if v == nil {
		return lua.LNil
	}
	if lval, ok := v.(lua.LValue); ok {
		return lval
	}

	val := reflect.ValueOf(v)
	valKind := val.Kind()
	switch valKind {
	case reflect.Map:
		if val.IsNil() {
			return lua.LNil
		}
		tbl := L.NewTable()
		iter := val.MapRange()
		for iter.Next() {
			key := iter.Key().Interface()
			value := iter.Value().Interface()
			tbl.RawSet(FromGoValue(L, key), FromGoValue(L, value))
		}
		return tbl
	case reflect.Slice:
		if val.IsNil() {
			return lua.LNil
		}
		fallthrough
	case reflect.Array:
		tbl := L.NewTable()
		for i := 0; i < val.Len(); i++ {
			value := val.Index(i).Interface()
			tbl.Append(FromGoValue(L, value))
		}
		return tbl
	case reflect.Struct:
		tbl := L.NewTable()
		t := reflect.TypeOf(v)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			key := ""
			tag, ok := field.Tag.Lookup(StructTagName)
			if ok {
				parts := strings.Split(tag, ",")
				key = parts[0]
				options := parts[1:]

				// special case to omit a field
				if key == "-" && len(options) == 0 {
					continue
				}

				// TODO: implement tag options
			}
			if key == "" {
				key = field.Name
			}
			value := val.Field(i)

			tbl.RawSetString(key, FromGoValue(L, value.Interface()))
		}
		return tbl

	case reflect.Pointer, reflect.Interface:
		return FromGoValue(L, val.Elem().Interface())
	}
	return luar.New(L, v)
}
