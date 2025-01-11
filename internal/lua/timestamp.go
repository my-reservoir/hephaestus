package lua

import (
	lua "github.com/yuin/gopher-lua"
	"reflect"
	"time"
)

var layouts = []string{
	time.DateTime, time.UnixDate, time.RFC1123, time.RFC822, time.RFC850,
}

type timeDescriptor struct{}

func (d *timeDescriptor) Type() reflect.Type {
	return reflect.TypeOf((*time.Time)(nil)).Elem()
}

func (d *timeDescriptor) Name() string {
	return "time"
}

func (d *timeDescriptor) FromLuaUserData(ud *lua.LUserData) interface{} {
	if v, ok := ud.Value.(*time.Time); ok {
		return v.Format(time.RFC3339)
	} else {
		return nil
	}
}

func checkAnyTimeLike(n int, L *lua.LState) *time.Time {
	switch L.CheckAny(n).Type() {
	case lua.LTNumber:
		t := time.UnixMilli(L.CheckInt64(n))
		return &t
	case lua.LTString:
		t, err := time.Parse(time.RFC3339, L.CheckString(n))
		if err != nil {
			for i, length := 0, len(layouts); i < length; i++ {
				if t, err = time.Parse(layouts[i], L.CheckString(n)); err == nil {
					break
				}
			}
		}
		if err != nil {
			L.ArgError(n, err.Error())
		}
		return &t
	default:
		L.ArgError(n, "unsupported timestamp type")
		fallthrough
	case lua.LTNil:
		epoch := time.Unix(0, 0)
		return &epoch
	case lua.LTUserData:
		return checkTime(n, L)
	}
}

func checkTime(n int, L *lua.LState) *time.Time {
	if v, ok := L.CheckUserData(n).Value.(*time.Time); ok {
		return v
	} else {
		return nil
	}
}

func checkAnyDurationLike(n int, L *lua.LState) *time.Duration {
	switch L.CheckAny(n).Type() {
	case lua.LTNumber:
		d := time.Duration(L.CheckInt64(n))
		return &d
	case lua.LTString:
		d, err := time.ParseDuration(L.CheckString(n))
		if err != nil {
			L.ArgError(n, err.Error())
		}
		return &d
	default:
		L.ArgError(n, "unsupported duration type")
		fallthrough
	case lua.LTNil:
		d := time.Duration(0)
		return &d
	case lua.LTUserData:
		return checkDuration(n, L)
	}
}

func checkDuration(n int, L *lua.LState) *time.Duration {
	if v, ok := L.CheckUserData(n).Value.(*time.Duration); ok {
		return v
	} else {
		return nil
	}
}

func RegisterTimestampType(L *lua.LState) []TypeDescriptor {
	mt := L.NewTypeMetatable("time")
	mtDuration := L.NewTypeMetatable("duration")
	L.SetGlobal("time", mt)
	L.SetGlobal("duration", mtDuration)
	L.SetField(mtDuration, "@type", lua.LString("duration"))
	L.SetField(mtDuration, "new", L.NewFunction(func(L *lua.LState) int {
		var d *time.Duration
		if L.GetTop() > 0 {
			d = checkAnyDurationLike(1, L)
		} else {
			tmp := time.Duration(0)
			d = &tmp
		}
		ud := L.NewUserData()
		ud.Value = d
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(checkDuration(1, L).String()))
		return 1
	}))
	L.SetField(mtDuration, "__len", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(*checkDuration(1, L)))
		return 1
	}))
	L.SetField(mtDuration, "__add", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyDurationLike(1, L), checkAnyDurationLike(2, L)
		result := time.Duration(*a + *b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__sub", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyDurationLike(1, L), checkAnyDurationLike(2, L)
		result := time.Duration(*a - *b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__mul", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyDurationLike(1, L), checkAnyDurationLike(2, L)
		result := time.Duration(*a * *b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__div", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyDurationLike(1, L), checkAnyDurationLike(2, L)
		result := time.Duration(*a / *b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__mod", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyDurationLike(1, L), checkAnyDurationLike(2, L)
		result := time.Duration(*a % *b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mtDuration, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"hours": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkDuration(1, L).Hours()))
			return 1
		},
		"minutes": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkDuration(1, L).Minutes()))
			return 1
		},
		"seconds": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkDuration(1, L).Seconds()))
			return 1
		},
		"us": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkDuration(1, L).Microseconds()))
			return 1
		},
		"ms": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkDuration(1, L).Milliseconds()))
			return 1
		},
	}))
	L.SetField(mt, "@type", lua.LString("time"))
	L.SetField(mt, "now", L.NewFunction(func(L *lua.LState) int {
		tmp := time.Now()
		ud := L.NewUserData()
		ud.Value = &tmp
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "new", L.NewFunction(func(L *lua.LState) int {
		var t *time.Time
		switch argc := L.GetTop(); argc {
		default:
			fallthrough
		case 0:
			tmp := time.Now()
			t = &tmp
		case 1:
			t = checkAnyTimeLike(1, L)
		case 2:
			tmp, err := time.Parse(L.CheckString(1), L.CheckString(2))
			if err != nil {
				L.ArgError(1, err.Error())
			}
			t = &tmp
		}
		ud := L.NewUserData()
		ud.Value = t
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "__len", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(checkTime(1, L).UnixMilli()))
		return 1
	}))
	L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(checkTime(1, L).Format(time.RFC3339)))
		return 1
	}))
	L.SetField(mt, "__eq", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyTimeLike(1, L), checkAnyTimeLike(2, L)
		L.Push(lua.LBool(a.Equal(*b)))
		return 1
	}))
	L.SetField(mt, "__lt", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyTimeLike(1, L), checkAnyTimeLike(2, L)
		L.Push(lua.LBool(a.Before(*b)))
		return 1
	}))
	L.SetField(mt, "__le", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyTimeLike(1, L), checkAnyTimeLike(2, L)
		L.Push(lua.LBool(a.Before(*b) || a.Equal(*b)))
		return 1
	}))
	// To eliminate ambiguity, we should forbid the automatic conversion between number to time & duration
	L.SetField(mt, "__add", L.NewFunction(func(L *lua.LState) int {
		var (
			t   time.Time
			d   time.Duration
			err error
		)
		// Assume that former is a duration
		switch L.CheckAny(1).Type() {
		case lua.LTString:
			d, err = time.ParseDuration(L.CheckString(1))
			if err == nil { // The assumption is true, so the latter is a time
				t = checkAnyTimeLike(2, L).Add(d)
				goto finished
			} else {
				goto falseAssumption
			}
		case lua.LTUserData:
			if v, ok := L.CheckUserData(1).Value.(*time.Duration); ok {
				t = checkAnyTimeLike(2, L).Add(*v)
				goto finished
			} else {
				goto falseAssumption
			}
		default:
			L.ArgError(1, "ambiguous parameter")
			L.Push(L.CheckAny(1))
			return 1
		}
	falseAssumption:
		// the former is a time indeed
		t = checkAnyTimeLike(1, L).Add(*checkAnyDurationLike(2, L))
	finished:
		ud := L.NewUserData()
		ud.Value = &t
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "__sub", L.NewFunction(func(L *lua.LState) int {
		a, b := checkAnyTimeLike(1, L), checkAnyTimeLike(2, L)
		result := a.Sub(*b)
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mtDuration)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"format": func(L *lua.LState) int {
			L.Push(lua.LString(checkTime(1, L).Format(L.CheckString(2))))
			return 1
		},
		"year": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Year()))
			return 1
		},
		"month": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Month()))
			return 1
		},
		"day": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Day()))
			return 1
		},
		"hour": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Hour()))
			return 1
		},
		"minute": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Minute()))
			return 1
		},
		"second": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Second()))
			return 1
		},
		"ns": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Nanosecond()))
			return 1
		},
		"weekday": func(L *lua.LState) int {
			L.Push(lua.LNumber(checkTime(1, L).Weekday()))
			return 1
		},
	}))
	return []TypeDescriptor{
		&timeDescriptor{},
	}
}

func init() {
	Register(RegisterTimestampType)
}
