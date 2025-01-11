package lua

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"reflect"
	"sync"
)

type GlobalThis struct {
	Args []interface{}
	Ret  []interface{}
}

var (
	globalThese sync.Map // map[*lua.LState]*GlobalThis
	types       = make(map[string]TypeDescriptor)
)

func loadGlobalThis(key *lua.LState) *GlobalThis {
	v, _ := globalThese.Load(key)
	val, _ := v.(*GlobalThis)
	return val
}

func loadGlobalThisWithOk(key *lua.LState) (*GlobalThis, bool) {
	v, ok := globalThese.Load(key)
	if !ok {
		return nil, false
	}
	val, _ := v.(*GlobalThis)
	return val, true
}

func deleteGlobalThis(key *lua.LState) {
	globalThese.Delete(key)
}

func storeGlobalThis(key *lua.LState, val *GlobalThis) {
	globalThese.Store(key, val)
}

type TypeDescriptor interface {
	Type() reflect.Type
	Name() string
	FromLuaUserData(*lua.LUserData) interface{}
}

func goType(val lua.LValue) interface{} {
	switch val.Type() {
	case lua.LTNumber:
		return int64(lua.LVAsNumber(val))
	case lua.LTBool:
		return lua.LVAsBool(val)
	case lua.LTString:
		return lua.LVAsString(val)
	case lua.LTUserData:
		v, _ := val.(*lua.LUserData)
		tbl, _ := v.Metatable.(*lua.LTable)
		descriptor, ok := types[tbl.RawGetString("@type").String()]
		if !ok {
			return nil
		}
		return descriptor.FromLuaUserData(v)
	case lua.LTFunction:
		v, _ := val.(*lua.LFunction)
		return v.String()
	case lua.LTChannel:
		fallthrough
	case lua.LTNil:
		fallthrough
	case lua.LTThread:
		return nil
	case lua.LTTable:
		ret := make(map[string]interface{})
		v, _ := val.(*lua.LTable)
		v.ForEach(func(key lua.LValue, value lua.LValue) {
			ret[key.String()] = goType(value)
		})
		return ret
	}
	return nil
}

func RegisterGlobalThis(L *lua.LState) {
	mt := L.NewTypeMetatable("this")
	L.SetGlobal("this", mt)
	L.SetField(mt, "argc", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(len(loadGlobalThis(L).Args)))
		return 1
	}))
	L.SetField(mt, "argv", L.NewFunction(func(L *lua.LState) int {
		this := loadGlobalThis(L)
		argc := len(this.Args)
		if paramCount := L.GetTop(); paramCount > 0 {
			for i := 1; i <= paramCount; i++ {
				if pos := L.CheckInt(i); pos >= 1 && pos <= argc {
					L.Push(luaType(this.Args[pos-1]))
				} else {
					L.ArgError(i, fmt.Sprintf("invalid index %d out of bound [1, %d]", pos, argc))
					L.Push(lua.LNil)
				}
			}
			argc = paramCount
		} else {
			for _, v := range this.Args {
				L.Push(luaType(v))
			}
		}
		return argc
	}))
	L.SetField(mt, "returns", L.NewFunction(func(L *lua.LState) int {
		argc := L.GetTop()
		ret := make([]interface{}, 0, argc)
		for i := 1; i <= argc; i++ {
			v := goType(L.CheckAny(i))
			ret = append(ret, v)
		}
		if this, ok := loadGlobalThisWithOk(L); !ok || this == nil {
			storeGlobalThis(L, &GlobalThis{})
		}
		loadGlobalThis(L).Ret = ret
		return 0
	}))
}
