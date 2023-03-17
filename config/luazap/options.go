package luazap

import "go.uber.org/zap/zapcore"

type Option interface {
	apply(*LuaZap)
}

type optionFunc func(*LuaZap)

func (f optionFunc) apply(lz *LuaZap) {
	f(lz)
}

const DefaultCallerKey string = "caller"

func WithCallerKey(key string) Option {
	return optionFunc(func(lz *LuaZap) {
		lz.CallerKey = key
	})
}

func WithCaller(enabled bool) Option {
	var key string
	if enabled {
		key = DefaultCallerKey
	} else {
		key = zapcore.OmitKey
	}
	return WithCallerKey(key)
}

func AddCaller() Option {
	return WithCaller(true)
}
