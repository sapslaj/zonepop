package luazap

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestSmokeTest(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.Hooks(func(e zapcore.Entry) error {
		expectedMsg := "test log message from Lua"
		if e.Message != expectedMsg {
			t.Errorf("expected message did not match: got %q, wanted %q", e.Message, expectedMsg)
		}
		expectedLevel := zap.InfoLevel
		if e.Level != expectedLevel {
			t.Errorf("expected level did not match: got %s, wanted %s", e.Level.String(), expectedLevel.String())
		}
		return nil
	})))
	state.PreloadModule("zap", NewLoader(logger))
	err := state.DoString(`
		local zap = require("zap")
		zap.info("test log message from Lua", {with="fields"})
	`)
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}
}

func TestSprint(t *testing.T) {
	tests := map[string]struct {
		do   string
		want string
	}{
		"string": {
			do:   `return require("zap").sprint("hunter2")`,
			want: "hunter2",
		},
		"int": {
			do:   `return require("zap").sprint(69)`,
			want: "69",
		},
		"float": {
			do:   `return require("zap").sprint(420.69)`,
			want: "420.69",
		},
		"multiple values": {
			do:   `return require("zap").sprint("foo", "bar", "baz", 69)`,
			want: "foobarbaz69",
		},
		"table": {
			do:   `return require("zap").sprint({foo = "bar"})`,
			want: `map[foo:bar]`,
		},
	}
	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			state := lua.NewState()
			defer state.Close()
			logger := zaptest.NewLogger(t)
			state.PreloadModule("zap", NewLoader(logger))

			err := state.DoString(tc.do)
			if err != nil {
				t.Fatalf("%s: failed to execute Lua %q: %v", desc, tc.do, err)
			}
			got := state.Get(-1).String()
			if got != tc.want {
				t.Errorf("%s: output did not match: got %q, wanted %q", desc, got, tc.want)
			}
		})
	}
}

func TestSprintf(t *testing.T) {
	tests := map[string]struct {
		do   string
		want string
	}{
		"string": {
			do:   `return require("zap").sprintf("before-%s-after", "during")`,
			want: "before-during-after",
		},
		"int": {
			do:   `return require("zap").sprintf("%d", 69)`,
			want: "69",
		},
		"float": {
			do:   `return require("zap").sprintf("%.2f", 420.69)`,
			want: "420.69",
		},
		"multiple values": {
			do:   `return require("zap").sprintf("%s-%s", "foo", "bar")`,
			want: "foo-bar",
		},
		"table": {
			do:   `return require("zap").sprintf("%v", {foo = "bar"})`,
			want: `map[foo:bar]`,
		},
	}
	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			state := lua.NewState()
			defer state.Close()
			logger := zaptest.NewLogger(t)
			state.PreloadModule("zap", NewLoader(logger))

			err := state.DoString(tc.do)
			if err != nil {
				t.Fatalf("%s: failed to execute Lua %q: %v", desc, tc.do, err)
			}
			got := state.Get(-1).String()
			if got != tc.want {
				t.Errorf("%s: output did not match: got %q, wanted %q", desc, got, tc.want)
			}
		})
	}
}

func TestFields(t *testing.T) {
	tests := map[string]struct {
		do     string
		fields map[string]any
		opts   []Option
	}{
		"no fields and no caller": {
			do:     `require("zap").info("no fields")`,
			fields: map[string]any{},
			opts: []Option{
				WithCaller(false),
			},
		},
		"no fields except caller": {
			do: `require("zap").info("no fields except caller")`,
			fields: map[string]any{
				"caller": "<string>:1",
			},
			opts: []Option{
				WithCaller(true),
			},
		},
		"with fields and no caller": {
			do: `require("zap").info("with fields and no caller", {foo="bar"})`,
			fields: map[string]any{
				"foo": "bar",
			},
			opts: []Option{
				WithCaller(false),
			},
		},
		"with fields and caller": {
			do: `require("zap").info("no fields except caller", {foo="bar"})`,
			fields: map[string]any{
				"foo":    "bar",
				"caller": "<string>:1",
			},
			opts: []Option{
				WithCaller(true),
			},
		},
		"with fields and caller using AddCaller": {
			do: `require("zap").info("no fields except caller", {foo="bar"})`,
			fields: map[string]any{
				"foo":    "bar",
				"caller": "<string>:1",
			},
			opts: []Option{
				AddCaller(),
			},
		},
		"with fields and custom caller key": {
			do: `require("zap").info("no fields except caller", {foo="bar"})`,
			fields: map[string]any{
				"foo":           "bar",
				"luazap_caller": "<string>:1",
			},
			opts: []Option{
				WithCallerKey("luazap_caller"),
			},
		},
		"with table in fields": {
			do: `require("zap").info("with table in fields", {foo={sub="bar",bar="baz"}})`,
			fields: map[string]any{
				"foo": map[any]any{
					"sub": "bar",
					"bar": "baz",
				},
			},
			opts: []Option{
				WithCaller(false),
			},
		},
		"with bool field": {
			do: `require("zap").info("with bool field", {foo=true})`,
			fields: map[string]any{
				"foo":    true,
				"caller": "<string>:1",
			},
		},
		"with number field": {
			do: `require("zap").info("with bool field", {foo=69})`,
			fields: map[string]any{
				"foo":    float64(69),
				"caller": "<string>:1",
			},
		},
	}
	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			state := lua.NewState()
			defer state.Close()
			core, logs := observer.New(zap.InfoLevel)
			logger := zap.New(core)
			state.PreloadModule("zap", NewLoader(logger, tc.opts...))

			err := state.DoString(tc.do)
			if err != nil {
				t.Fatalf("%s: failed to execute Lua %q: %v", desc, tc.do, err)
			}
			if logs.Len() == 0 {
				t.Errorf("%s: no logs received", desc)
			}
			fields := logs.All()[logs.Len()-1].ContextMap()
			if diff := cmp.Diff(fields, tc.fields); diff != "" {
				t.Errorf("%s: mismatch:\n%s", desc, diff)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core, zap.WithFatalHook(zapcore.WriteThenNoop))
	state.PreloadModule("zap", NewLoader(logger))
	err := state.DoString(`
		local zap = require("zap")
		zap.debug("debug")
		zap.info("info")
		zap.warn("warn")
		zap.error("error")
		-- zap.fatal("fatal")
	`)
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}
	for _, levelString := range []string{
		"debug",
		"error",
		// "fatal", fatal testing disabled because zapcore.WriteThenNoop isn't working
		"info",
		"warn",
	} {
		t.Run(levelString, func(t *testing.T) {
			level, err := zapcore.ParseLevel(levelString)
			if err != nil {
				t.Fatalf("failed to parse zap level %q: %v", levelString, err)
			}
			filteredLogs := logs.FilterLevelExact(level).All()
			if len(filteredLogs) != 1 {
				t.Fatalf("len(logs[level==%s]) != 1 (got %d)", levelString, len(filteredLogs))
			}
			if filteredLogs[0].Message != levelString {
				t.Errorf("log for level %s did not match expected string: %v", levelString, filteredLogs[0])
			}
		})
	}
}
