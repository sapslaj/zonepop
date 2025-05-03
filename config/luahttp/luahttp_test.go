package luahttp

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestRequest(t *testing.T) {
	state := lua.NewState()
	defer state.Close()
	state.PreloadModule("http", NewLoader())
	err := state.DoFile("test_lua/test_request.lua")
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}
	tests := state.Get(-1).(*lua.LTable)
	tests.ForEach(func(key, value lua.LValue) {
		t.Run(key.String(), func(t *testing.T) {
			co, _ := state.NewThread()
			tc := value.(*lua.LFunction)
			st, err, _ := state.Resume(co, tc)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if st != lua.ResumeOK {
				t.Fatalf("lua execution is not ok; state=%v", st)
			}
		})
	})
}
