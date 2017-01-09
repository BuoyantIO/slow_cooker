package scripting

import (
	"errors"
	"net/url"

	"github.com/yuin/gluamapper"
	"github.com/yuin/gopher-lua"
)

type LReqData struct {
	Method string
	Url    string
	Host   string
	Body   string
}

func (reqData *LReqData) GetUrl() (*url.URL, error) {
	return url.Parse(reqData.Url)
}

func (reqData *LReqData) MustGetUrl() *url.URL {
	u, err := reqData.GetUrl()
	if err != nil {
		panic(err)
	}
	return u
}

func (reqData *LReqData) GetBody() []byte {
	return []byte(reqData.Body)
}

// NewDataGenerator returns a function that calls the slow_cooker.generate_data function
// and retrieves its value from the stack.
func NewDataGenerator(luaPool *lStatePool) func(*LReqData, uint64) error {
	return func(reqData *LReqData, reqID uint64) error {
		l := luaPool.Get()
		defer luaPool.Put(l)
		if err := l.CallByParam(lua.P{
			Fn:      l.GetField(l.GetGlobal("slow_cooker"), "generate_data"),
			NRet:    1,
			Protect: true,
		}, lua.LString(reqData.Method), lua.LString(reqData.Url), lua.LString(reqData.Host), lua.LNumber(float64(reqID))); err != nil {
			return err
		}
		defer l.Pop(1)
		if ret, ok := l.Get(-1).(*lua.LTable); ok {
			if err := gluamapper.Map(ret, reqData); err != nil {
				return err
			}
			return nil
		}
		return errors.New("Unable to get return value from lua stack")
	}
}
