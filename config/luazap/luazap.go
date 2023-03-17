package luazap

import (
	"fmt"
	"math"
	"strings"

	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LuaZap struct {
	Logger    *zap.Logger
	CallerKey string
}

func NewLoader(logger *zap.Logger, opts ...Option) func(*lua.LState) int {
	return NewLuaZap(logger, opts...).Loader
}

func NewLuaZap(logger *zap.Logger, opts ...Option) *LuaZap {
	lz := &LuaZap{
		Logger:    logger,
		CallerKey: DefaultCallerKey,
	}
	for _, o := range opts {
		o.apply(lz)
	}
	return lz
}

func (lz *LuaZap) Loader(L *lua.LState) int {
	exports := make(map[string]lua.LGFunction)
	funcs := make(map[string]lua.LGFunction)
	for _, level := range []string{
		"debug",
		"error",
		"fatal",
		"info",
		"warn",
	} {
		funcs[level] = lz.makeLogFunc(level)
	}

	for f, fn := range funcs {
		exports[f] = fn
	}

	exports["sprint"] = lz.makeSprintFunc()
	exports["sprintf"] = lz.makeSprintfFunc()

	mod := L.SetFuncs(L.NewTable(), exports)
	L.Push(mod)
	return 1
}

func (lz *LuaZap) ToGoValue(lv lua.LValue) any {
	value := gluamapper.ToGoValue(lv, gluamapper.Option{
		NameFunc: gluamapper.Id,
	})
	// Hack to make ints work
	if floatValue, ok := value.(float64); ok {
		if math.Round(floatValue) == floatValue {
			value = int64(floatValue)
		}
	}
	return value
}

func (lz *LuaZap) makeSprintFunc() func(L *lua.LState) int {
	return func(L *lua.LState) int {
		values := make([]any, 0)
		for i := 1; i <= L.GetTop(); i++ {
			values = append(values, lz.ToGoValue(L.Get(i)))
		}
		L.Push(lua.LString(fmt.Sprint(values...)))
		return 1
	}
}

func (lz *LuaZap) makeSprintfFunc() func(L *lua.LState) int {
	return func(L *lua.LState) int {
		format := L.CheckString(1)
		values := make([]any, 0)
		for i := 2; i <= L.GetTop(); i++ {
			values = append(values, lz.ToGoValue(L.Get(i)))
		}
		L.Push(lua.LString(fmt.Sprintf(format, values...)))
		return 1
	}
}

func (lz *LuaZap) TableToZapFields(lt *lua.LTable) []zapcore.Field {
	fields := make([]zapcore.Field, 0)
	if lt == nil {
		return fields
	}

	var goFields map[string]any
	fieldMapper := gluamapper.NewMapper(gluamapper.Option{
		NameFunc: gluamapper.Id,
	})
	err := fieldMapper.Map(lt, &goFields)
	if err != nil {
		// TODO: better error handling
		panic(err)
	}
	for k, v := range goFields {
		var field zapcore.Field
		switch t := v.(type) {
		case bool:
			field = zap.Bool(k, t)
		case string:
			field = zap.String(k, t)
		case float64:
			field = zap.Float64(k, t)
		default:
			field = zap.Any(k, v)
		}
		fields = append(fields, field)
	}

	return fields
}

func (lz *LuaZap) makeLogFunc(level string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		msg := L.CheckString(1)
		lFields := L.OptTable(2, L.NewTable())
		fields := make([]zapcore.Field, 0)
		logger := lz.Logger
		if lz.CallerKey != zapcore.OmitKey {
			caller := strings.TrimSuffix(L.Where(-1), ":")
			logger = lz.Logger.With(zap.String(lz.CallerKey, caller))
		}

		fields = append(fields, lz.TableToZapFields(lFields)...)

		switch level {
		case "debug":
			logger.Debug(msg, fields...)
		case "error":
			logger.Error(msg, fields...)
		case "fatal":
			logger.Fatal(msg, fields...)
		case "info":
			logger.Info(msg, fields...)
		case "warn":
			logger.Warn(msg, fields...)
		default:
			panic(fmt.Errorf("unsupported log level %s", level))
		}
		return 0
	}
}
