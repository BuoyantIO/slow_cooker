package scripting

import (
	"github.com/yuin/gopher-lua"
)

// NewModuleLoader creates a new module loader that registers the given functions
// to the lua module's exported lua table. This can be used to expose Go functions
// to lua. See https://github.com/yuin/gopher-lua#calling-go-from-lua
func NewModuleLoader(exports map[string]lua.LGFunction) func(*lua.LState) int {
	return func(l *lua.LState) int {
		// register functions to the module's exported lua table
		// local slow_cooker = require("slow_cooker")
		mod := l.SetFuncs(l.NewTable(), exports)
		l.Push(mod)
		return 1
	}
}
