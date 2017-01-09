package scripting

import (
	"sync"

	gopherJson "github.com/layeh/gopher-json"
	"github.com/yuin/gopher-lua"
)

var DefaultModuleLoaders = map[string]func(*lua.LState) int{
	"slow_cooker": NewModuleLoader(map[string]lua.LGFunction{}),
	"gopherJson":  gopherJson.Loader,
}

// A very slightly modified version of the lStatePool from the gopher-lua readme
// https://github.com/yuin/gopher-lua#the-lstate-pool-pattern
type lStatePool struct {
	m             sync.Mutex
	saved         []*lua.LState
	script        string
	moduleLoaders map[string]func(*lua.LState) int
}

func NewLStatePool(script string, capacity int, moduleLoaders map[string]func(*lua.LState) int) *lStatePool {
	return &lStatePool{
		saved:         make([]*lua.LState, 0, capacity),
		script:        script,
		moduleLoaders: moduleLoaders,
	}
}

func (pl *lStatePool) Get() *lua.LState {
	pl.m.Lock()
	defer pl.m.Unlock()
	n := len(pl.saved)
	if n == 0 {
		return pl.New()
	}
	x := pl.saved[n-1]
	pl.saved = pl.saved[0 : n-1]
	return x
}

func (pl *lStatePool) New() *lua.LState {
	l := lua.NewState()
	for modName, loader := range pl.moduleLoaders {
		l.PreloadModule(modName, loader)
	}
	if err := l.DoFile(pl.script); err != nil {
		panic(err)
	}
	return l
}

func (pl *lStatePool) Put(l *lua.LState) {
	pl.m.Lock()
	defer pl.m.Unlock()
	pl.saved = append(pl.saved, l)
}

func (pl *lStatePool) Shutdown() {
	for _, l := range pl.saved {
		l.Close()
	}
}
