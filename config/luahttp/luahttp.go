package luahttp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"

	"github.com/sapslaj/zonepop/pkg/gluamapper"
)

// TODO: support auth and proxies
type Request struct {
	URL                string
	Method             string
	Body               string
	JSON               any
	Headers            map[string][]string
	TimeoutSeconds     int
	InsecureSkipVerify bool
	// AllowRedirects     bool  // FIXME
}

type LuaHTTP struct{}

func NewLuaHTTP() *LuaHTTP {
	return &LuaHTTP{}
}

func (lhttp *LuaHTTP) Loader(L *lua.LState) int {
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"request": lhttp.L_Request,
	})
	L.Push(mod)
	return 1
}

func NewLoader() func(*lua.LState) int {
	return NewLuaHTTP().Loader
}

func (lhttp *LuaHTTP) JSONUnmarshal(L *lua.LState, v any) lua.LValue {
	switch val := v.(type) {
	case []any:
		tbl := L.NewTable()
		for _, item := range val {
			tbl.Append(lhttp.JSONUnmarshal(L, item))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for key, value := range val {
			tbl.RawSetString(key, lhttp.JSONUnmarshal(L, value))
		}
		return tbl
	}
	return luar.New(L, v)
}

func (lhttp *LuaHTTP) L_Request(L *lua.LState) int {
	reqTbl := L.CheckTable(1)
	var req Request
	err := gluamapper.Map(reqTbl, &req)
	if err != nil {
		L.RaiseError("error making request: %v", err)
		return 0
	}

	ctx := context.Background()
	if req.TimeoutSeconds != 0 {
		var cancelCtx context.CancelFunc
		ctx, cancelCtx = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancelCtx()
	}

	if req.Method == "" {
		req.Method = "GET"
	}

	var reqBody io.Reader
	if req.JSON != nil {
		data, err := json.Marshal(req.JSON)
		if err != nil {
			L.RaiseError("error making request: %v", err)
			return 0
		}
		reqBody = bytes.NewReader(data)
	} else if req.Body != "" {
		reqBody = strings.NewReader(req.Body)
	}

	clientTransport := &http.Transport{}

	if req.InsecureSkipVerify {
		clientTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client := &http.Client{
		Transport: clientTransport,
	}

	httpRequest, err := http.NewRequestWithContext(ctx, req.Method, req.URL, reqBody)
	if err != nil {
		L.RaiseError("error making request: %v", err)
		return 0
	}
	if req.Headers == nil {
		httpRequest.Header = http.Header{}
	} else {
		httpRequest.Header = http.Header(req.Headers)
	}

	// auto-set "Content-Type" header
	// we do this were so we can use `http.Header` methods
	if req.JSON != nil && httpRequest.Header.Get("Content-Type") == "" {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		L.RaiseError("error making request: %v", err)
		return 0
	}

	var resBody string
	if httpResponse.Body != nil {
		resData, err := io.ReadAll(httpResponse.Body)
		defer httpResponse.Body.Close()
		if err != nil {
			L.RaiseError("error making request: %v", err)
			return 0
		}
		resBody = string(resData)
	}

	headers := map[string]*lua.LTable{}
	for key, values := range httpResponse.Header {
		t := L.NewTable()
		for _, value := range values {
			t.Append(lua.LString(value))
		}
		headers[key] = t
	}

	res := L.NewTable()
	res.RawSetString("status_code", luar.New(L, httpResponse.StatusCode))
	res.RawSetString("headers", luar.New(L, headers))
	res.RawSetString("body", luar.New(L, resBody))
	res.RawSetString("json", luar.New(L, func(LL *luar.LState) int {
		var raw any
		err := json.Unmarshal([]byte(resBody), &raw)
		if err != nil {
			LL.RaiseError("error parsing request JSON: %v", err)
			return 0
		}

		LL.Push(lhttp.JSONUnmarshal(LL.LState, raw))

		return 1
	}))

	L.Push(res)

	return 1
}
