package scripting

import (
	"fmt"
	"github.com/yuin/gopher-lua"
	"net/url"
)

func NewDataGenerator(luaPool *lStatePool) func(string, *url.URL, string, uint64) []byte {
	return func(method string, url *url.URL, host string, reqID uint64) []byte {
		l := luaPool.Get()
		defer luaPool.Put(l)
		if err := l.CallByParam(lua.P{
			Fn:      l.GetField(l.GetGlobal("slow_cooker"), "generate_data"),
			NRet:    1,
			Protect: true,
		}, lua.LString(method), lua.LString(url.String()), lua.LString(host), lua.LNumber(float64(reqID))); err != nil {
			panic(err)
		}
		defer l.Pop(1)
		if ret, ok := l.Get(-1).(lua.LString); ok {
			s := fmt.Sprint(ret)
			return []byte(s)
		}
		panic("Unable to get return value from lua stack")
	}

}
