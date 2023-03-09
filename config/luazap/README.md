# luazap

[GopherLua](https://github.com/yuin/gopher-lua) wrapper for [zap](https://github.com/uber-go/zap).

Currently embedded as part of [ZonePop](https://github.com/sapslaj/zonepop) so API stability is not guaranteed.

This lib only uses regular `Logger` and not `SugaredLogger`. In addition, not all of the features of zap are supported yet.

## Quick Start

```go
package main

import (
	"github.com/sapslaj/zonepop/config/luazap"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

func main() {
	// You can create a logger and use it like normal
	logger := zap.Must(zap.NewProduction())
	logger.Info("Startup")

	// Start up Lua
	state := lua.NewState()
	defer state.Close()

	// Load luazap
	state.PreloadModule(
		"zap",

		// Is it recommended to disable zap's built-in caller tracking because
		// luazap provides its own.
		luazap.NewLoader(logger.WithOptions(zap.WithCaller(false))),
	)

	state.DoFile("main.lua")
}
```

```lua
local log = require("zap")

log.info("info stuff")
log.warn("fields are supported", {extra_fields = "here"})
```

```shell
$ go run main.go
{"level":"info","ts":1678325793.447191,"caller":"main.go:12","msg":"Startup"}
{"level":"info","ts":1678325793.4476,"msg":"info stuff","caller":"main.lua:3"}
{"level":"warn","ts":1678325793.4476254,"msg":"fields are supported","caller":"main.lua:4","extra_fields":"here"}
```

## Lua Functions

### `debug`

`zap.debug(msg [, fields])`

Arguments:
| | | |
| --- | --- | --- |
| `msg` | string | Message to write at debug level |
| `fields` | table | Optional fields to include in table format with string key, value pairs |

### `error`

`zap.error(msg [, fields])`

Arguments:
| | | |
| --- | --- | --- |
| `msg` | string | Message to write at error level |
| `fields` | table | Optional fields to include in table format with string key, value pairs |

### `fatal`

`zap.fatal(msg [, fields])`

Arguments:
| | | |
| --- | --- | --- |
| `msg` | string | Message to write at fatal level |
| `fields` | table | Optional fields to include in table format with string key, value pairs |

### `info`

`zap.info(msg [, fields])`

Arguments:
| | | |
| --- | --- | --- |
| `msg` | string | Message to write at info level |
| `fields` | table | Optional fields to include in table format with string key, value pairs |

### `warn`

`zap.warn(msg [, fields])`

Arguments:
| | | |
| --- | --- | --- |
| `msg` | string | Message to write at warn level |
| `fields` | table | Optional fields to include in table format with string key, value pairs |

### `sprint`

`zap.sprint([... value]) -> string`

Helper function for calling Go `fmt.Sprint` from Lua. This is helpful for getting the formatting of labels in a way that zap likes.

Arguments:
| | | |
| --- | --- | --- |
| `value` | any | 1 or more values to call `fmt.Sprint` with |

Returns:
| | |
| --- | --- |
| string | Result from `fmt.Sprint` |

### `sprintf`

`zap.sprintf(format [, ... value]) -> string`

Helper function for calling Go `fmt.Sprintf` from Lua. This is helpful for getting the formatting of labels in a way that zap likes.

Arguments:
| | | |
| --- | --- | --- |
| `format` | string | Format string for `fmt.Sprintf` call |
| `value` | any | 1 or more values to call `fmt.Sprintf` with |

Returns:
| | |
| --- | --- |
| string | Result from `fmt.Sprintf` |
