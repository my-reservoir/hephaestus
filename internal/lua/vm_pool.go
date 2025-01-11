package lua

import (
	lua "github.com/yuin/gopher-lua"
	"sync"
)

type VM = *lua.LState
type RegisterFunc = func(VM) []TypeDescriptor
type VMPool interface {
	Register(...RegisterFunc)
	New() VM
	Put(VM)
	Get() VM
	Shutdown()
	RunString(string, ...interface{}) ([]interface{}, error)
}

var registeredFunc = make([]RegisterFunc, 0, 8)

type vmPool struct {
	Options    *lua.Options
	m          sync.Mutex
	saved      []*lua.LState
	registered []RegisterFunc
	chanWait   chan struct{}
	limit      int
	waiting    int
	running    int
}

var (
	defaultPool = NewVMPool()
)

func Pool() VMPool {
	return defaultPool
}

func NewVMPool() VMPool {
	p := &vmPool{
		Options:    &lua.Options{},
		saved:      make([]*lua.LState, 0, 8),
		registered: make([]RegisterFunc, 0, len(registeredFunc)+8),
		limit:      256,
		chanWait:   make(chan struct{}),
	}
	p.registered = append(p.registered, registeredFunc...)
	return p
}

func Register(fun ...RegisterFunc) {
	registeredFunc = append(registeredFunc, fun...)
	defaultPool.Register(fun...)
}

func (p *vmPool) Register(fun ...RegisterFunc) {
	p.registered = append(p.registered, fun...)
	if len(p.saved) == 0 {
		p.saved = append(p.saved, p.New())
	}
	for _, f := range fun {
		desc := f(p.saved[0])
		if len(desc) > 0 {
			for _, d := range desc {
				if _, ok := types[d.Name()]; !ok {
					types[d.Name()] = d
				}
			}
		}
	}
}

func (p *vmPool) New() VM {
	vm := lua.NewState(*p.Options)
	RegisterGlobalThis(vm)
	for _, r := range p.registered {
		r(vm)
	}
	return vm
}

func (p *vmPool) Put(vm VM) {
	p.m.Lock()
	p.saved = append(p.saved, vm)
	p.m.Unlock()
	p.running--
	if p.waiting > 0 {
		p.chanWait <- struct{}{}
		p.waiting--
	}
}

func (p *vmPool) Get() (vm VM) {
	count := len(p.saved)
	overall := count + p.running
	if overall >= p.limit {
		p.waiting++
		<-p.chanWait
	}
	if count == 0 {
		p.running++
		return p.New()
	} else {
		vm = p.saved[count-1]
		p.m.Lock()
		defer p.m.Unlock()
		p.running++
		p.saved = p.saved[:count-1]
		return
	}
}

func (p *vmPool) RunString(str string, args ...interface{}) (ret []interface{}, e error) {
	vm := p.Get()
	defer func() {
		storeGlobalThis(vm, nil)
		if err := recover(); err != nil {
			e, _ = err.(error)
		}
	}()
	defer p.Put(vm)
	storeGlobalThis(vm, &GlobalThis{Args: args})
	e = vm.DoString(str)
	ret = loadGlobalThis(vm).Ret
	return
}

func (p *vmPool) Shutdown() {
	for _, vm := range p.saved {
		vm.Close()
	}
	clear(p.saved)
}
