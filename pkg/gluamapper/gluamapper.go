// gluamapper provides an easy way to map GopherLua tables to Go structs.
package gluamapper

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/yuin/gopher-lua"
)

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
	if config.TagName == "" {
		config.TagName = "gluamapper"
	}
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
