---@class HTTPRequest
---@field url string
---@field method? string
---@field body? string
---@field json? table
---@field headers? { [string]: string[] }
---@field timeout_seconds? number
---@field insecure_skip_verify? boolean

---@class HTTPResponse
---@field status_code number
---@field headers { [string]: string[] }
---@field body string
---@field json fun(): boolean

---@class http
---@field request fun(opts: HTTPRequest): HTTPResponse

local http = require("http") --[[@as http]]

return {
  ["GET request"] = function ()
    local res = http.request({
      method = "GET",
      url = "https://httpbingo.org/get",
    })
    if res.status_code ~= 200 then
      error(string.format("unexpected status code (want 200, got %d)", res.status_code))
    end
  end,
  ["implicit GET"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/get",
    })
    if res.status_code ~= 200 then
      error(string.format("unexpected status code (want 200, got %d)", res.status_code))
    end
  end,
  ["POST request"] = function ()
    local res = http.request({
      method = "POST",
      url = "https://httpbingo.org/post",
    })
    if res.status_code ~= 200 then
      error(string.format("unexpected status code (want 200, got %d)", res.status_code))
    end
  end,
  ["200 status code"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/status/200",
    })
    if res.status_code ~= 200 then
      error(string.format("unexpected status code (want 200, got %d)", res.status_code))
    end
  end,
  ["400 status code"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/status/400",
    })
    if res.status_code ~= 400 then
      error(string.format("unexpected status code (want 400, got %d)", res.status_code))
    end
  end,
  ["500 status code"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/status/500",
    })
    if res.status_code ~= 500 then
      error(string.format("unexpected status code (want 500, got %d)", res.status_code))
    end
  end,
  ["headers"] = function()
    local res = http.request({
      url = "https://httpbingo.org/headers",
      headers = {
        ["X-Foo-Bar"] = {"baz"},
      }
    })
    if res.headers["Access-Control-Allow-Origin"][1] ~= "*" then
      error(string.format(
        "unexpected \"Access-Control-Allow-Origin\" header value (want \"*\", got %q)",
        res.headers["Access-Control-Allow-Origin"]
      ))
    end
    local json = res.json()
    if json["headers"]["X-Foo-Bar"][1] ~= "baz" then
      error(string.format(
        "unexpected \"X-Foo-Bar\" header value (want \"baz\", got %q)",
        json["headers"]["X-Foo-Bar"]
      ))
    end
  end,
  ["plain-text request body"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/anything",
      method = "POST",
      body = "this is raw data",
      headers = {
        ["Content-Type"] = {"text/plain"},
      },
    })
    if res.json()["data"] ~= "this is raw data" then
      -- FIXME: this error message is terrible
      error("raw data not echoed from server")
    end
  end,
  ["plain-text response body"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/encoding/utf8",
    })
    if not string.find(res.body, "Unicode Demo") then
      error("didn't find expected unicode string in response body")
    end
  end,
  ["JSON"] = function ()
    local res = http.request({
      method = "GET",
      url = "https://httpbingo.org/anything",
      json = {
        str = "foo",
        int = 69,
        bool = true,
        null = nil,
        obj = {
          ["foo"] = "bar",
        },
        arr = {
          "foo",
          "bar",
          "baz",
        },
      },
    })
    local json = res.json()["json"]
    pairs(json) -- assert that it is a table
    if json.str ~= "foo" then
      error(string.format("unexpected json.str value (want \"foo\", got %q)", json.str))
    end
    if json.int ~= 69 then
      error(string.format("unexpected json.int value (want 69, got %q)", json.str))
    end
    if json.bool ~= true then
      error(string.format("unexpected json.bool value (want true, got %q)", json.bool))
    end
    if json.null ~= nil then
      error(string.format("unexpected json.null value (want nil, got %q)", json.null))
    end
    if json.obj.foo ~= "bar" then
      error(string.format("unexpected json.obj.foo value (want \"bar\", got %q)", json.obj.foo))
    end
    if json.arr[1] ~= "foo" then
      error(string.format("unexpected json.arr[1] value (want \"foo\", got %q)", json.arr[1]))
    end
    if json.arr[2] ~= "bar" then
      error(string.format("unexpected json.arr[2] value (want \"bar\", got %q)", json.arr[2]))
    end
    if json.arr[3] ~= "baz" then
      error(string.format("unexpected json.arr[3] value (want \"baz\", got %q)", json.arr[3]))
    end
  end,
  ["TLS cert verification"] = function()
    if pcall(function ()
      http.request({
        url = "https://expired.badssl.com/"
      })
    end) then
      error("TLS verification should have failed with an error but did not")
    end
  end,
  ["TLS insecure skip verify"] = function ()
    local res = http.request({
      url = "https://expired.badssl.com/",
      insecure_skip_verify = true,
    })
    if not string.find(res.body, "expired.badssl.com") then
      error("did not get expected response body")
    end
  end,
  ["timeout"] = function ()
    local res = http.request({
      url = "https://httpbingo.org/delay/1",
      insecure_skip_verify = true,
      timeout_seconds = 10,
    })
    if res.status_code ~= 200 then
      error(string.format("unexpected status code (want 200, got %d)", res.status_code))
    end

    if pcall(function ()
      http.request({
        url = "https://httpbingo.org/delay/3",
        insecure_skip_verify = true,
        timeout_seconds = 1,
      })
    end) then
      error("expected error on timeout but got none")
    end
  end,
}
