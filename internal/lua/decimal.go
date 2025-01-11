package lua

import (
	"fmt"
	"github.com/shopspring/decimal"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	"math/rand"
	"reflect"
)

var (
	constHalf, _ = decimal.NewFromString(".5")
)

type decimalDescriptor struct{}

func (d *decimalDescriptor) Type() reflect.Type {
	return reflect.TypeOf((*decimal.Decimal)(nil)).Elem()
}
func (d *decimalDescriptor) Name() string {
	return "decimal"
}
func (d *decimalDescriptor) FromLuaUserData(dat *lua.LUserData) interface{} {
	if v, ok := dat.Value.(*decimal.Decimal); ok {
		return v.String()
	} else {
		return nil
	}
}

func checkDecimal(n int, L *lua.LState) *decimal.Decimal {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*decimal.Decimal); ok {
		return v
	}
	L.ArgError(n, fmt.Sprintf("decimal expected, got %v", ud.Type()))
	ret := decimal.NewFromInt(0)
	return &ret
}

func checkAnyDecimalLike(n int, L *lua.LState) *decimal.Decimal {
	switch L.CheckAny(n).Type() {
	case lua.LTNumber:
		ret := decimal.NewFromInt(L.CheckInt64(n))
		return &ret
	case lua.LTString:
		ret, err := decimal.NewFromString(L.CheckString(n))
		if err != nil {
			L.ArgError(n, err.Error())
		}
		return &ret
	case lua.LTUserData:
		return checkDecimal(n, L)
	default:
		L.ArgError(n, "unsupported decimal type")
		ret := decimal.NewFromInt(0)
		return &ret
	}
}

func performDecimalOp(
	L *lua.LState, mt *lua.LTable,
	op func(decimal.Decimal, decimal.Decimal) decimal.Decimal) int {
	if argc := L.GetTop(); argc >= 2 {
		result := *checkAnyDecimalLike(1, L)
		for i := 2; i <= argc; i++ {
			result = op(result, *checkAnyDecimalLike(i, L))
		}
		ud := L.NewUserData()
		ud.Value = &result
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}
	ud := L.CheckUserData(1)
	L.SetMetatable(ud, mt)
	L.Push(ud)
	return 1
}

func performDecimalCmpOp(L *lua.LState, mt *lua.LTable,
	op func(decimal.Decimal, decimal.Decimal) bool) int {
	if argc := L.GetTop(); argc == 2 {
		a, b := checkAnyDecimalLike(1, L), checkAnyDecimalLike(2, L)
		L.Push(lua.LBool(op(*a, *b)))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func performDecimalMathOp(L *lua.LState, mt *lua.LTable,
	op func(decimal.Decimal) decimal.Decimal) int {
	if argc := L.GetTop(); argc == 1 {
		a := checkAnyDecimalLike(1, L)
		res := op(*a)
		ud := L.NewUserData()
		ud.Value = &res
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func performDecimalRoundingOp(
	L *lua.LState, mt *lua.LTable,
	op func(decimal.Decimal, int32) decimal.Decimal) int {
	if argc := L.GetTop(); argc == 2 {
		a, precision := checkDecimal(1, L), L.CheckInt64(2)
		res := op(*a, int32(precision))
		ud := L.NewUserData()
		ud.Value = &res
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func RegisterDecimalType(L *lua.LState) []TypeDescriptor {
	mt := L.NewTypeMetatable("decimal")
	L.SetGlobal("decimal", mt)
	L.SetField(mt, "@type", lua.LString("decimal"))
	L.SetField(mt, "new", L.NewFunction(func(L *lua.LState) int {
		if argc := L.GetTop(); argc >= 1 {
			for i := 1; i <= argc; i++ {
				ud := L.NewUserData()
				ud.Value = checkAnyDecimalLike(i, L)
				L.SetMetatable(ud, mt)
				L.Push(ud)
			}
			return argc
		} else {
			ud := L.NewUserData()
			ret := decimal.NewFromInt(0)
			ud.Value = &ret
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		}
	}))
	udPi, udE := L.NewUserData(), L.NewUserData()
	pi, _ := decimal.NewFromString("3.141592653589793238462643383279502884197169399375105820974944592")
	e, _ := decimal.NewFromString("2.718281828459045235360287471352662497757247093699959574966967628")
	udPi.Value = &pi
	udE.Value = &e
	L.SetMetatable(udPi, mt)
	L.SetMetatable(udE, mt)
	L.SetField(mt, "pi", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() == 1 {
			rounded := pi.Round(int32(L.CheckInt64(1)))
			ud := L.NewUserData()
			ud.Value = &rounded
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		}
		L.Push(udPi)
		return 1
	}))
	L.SetField(mt, "e", L.NewFunction(func(L *lua.LState) int {
		if L.GetTop() == 1 {
			rounded := e.Round(int32(L.CheckInt64(1)))
			ud := L.NewUserData()
			ud.Value = &rounded
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		}
		L.Push(udE)
		return 1
	}))
	L.SetField(mt, "min", L.NewFunction(func(L *lua.LState) int {
		argc := L.GetTop()
		if argc < 2 {
			L.ArgError(argc, fmt.Sprintf("not enough arguments, at least 2 but %d is provided", argc))
			L.Push(lua.LNil)
			return 1
		}
		decimals := make([]decimal.Decimal, 0, argc)
		for i := 1; i <= argc; i++ {
			decimals = append(decimals, *checkAnyDecimalLike(i, L))
		}
		dec := decimal.Min(decimals[0], decimals[1:]...)
		ud := L.NewUserData()
		ud.Value = &dec
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "max", L.NewFunction(func(L *lua.LState) int {
		argc := L.GetTop()
		if argc < 2 {
			L.ArgError(argc, fmt.Sprintf("not enough arguments, at least 2 but %d is provided", argc))
			L.Push(lua.LNil)
			return 1
		}
		decimals := make([]decimal.Decimal, 0, argc)
		for i := 1; i <= argc; i++ {
			decimals = append(decimals, *checkAnyDecimalLike(i, L))
		}
		dec := decimal.Max(decimals[0], decimals[1:]...)
		ud := L.NewUserData()
		ud.Value = &dec
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "random", L.NewFunction(func(L *lua.LState) int {
		var dec decimal.Decimal
		switch argc := L.GetTop(); argc {
		case 1:
			dec = decimal.NewFromInt(rand.Int63n(L.CheckInt64(1)))
		case 2:
			from, to := L.CheckInt64(1), L.CheckInt64(2)
			dec = decimal.NewFromInt(from + rand.Int63n(to-from))
		default:
			dec = decimal.NewFromFloat(rand.Float64())
		}
		ud := L.NewUserData()
		ud.Value = &dec
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(checkDecimal(1, L).String()))
		return 1
	}))
	L.SetField(mt, "__len", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(checkDecimal(1, L).NumDigits()))
		return 1
	}))
	L.SetField(mt, "__add", L.NewFunction(func(L *lua.LState) int {
		return performDecimalOp(L, mt, decimal.Decimal.Add)
	}))
	L.SetField(mt, "__sub", L.NewFunction(func(L *lua.LState) int {
		return performDecimalOp(L, mt, decimal.Decimal.Sub)
	}))
	L.SetField(mt, "__mul", L.NewFunction(func(L *lua.LState) int {
		return performDecimalOp(L, mt, decimal.Decimal.Mul)
	}))
	L.SetField(mt, "__div", L.NewFunction(func(L *lua.LState) int {
		return performDecimalOp(L, mt, decimal.Decimal.Div)
	}))
	L.SetField(mt, "__unm", L.NewFunction(func(L *lua.LState) int {
		dec := checkDecimal(1, L).Neg()
		ud := L.NewUserData()
		ud.Value = &dec
		L.SetMetatable(ud, mt)
		L.Push(ud)
		return 1
	}))
	L.SetField(mt, "__eq", L.NewFunction(func(L *lua.LState) int {
		return performDecimalCmpOp(L, mt, decimal.Decimal.Equal)
	}))
	L.SetField(mt, "__lt", L.NewFunction(func(L *lua.LState) int {
		return performDecimalCmpOp(L, mt, decimal.Decimal.LessThan)
	}))
	L.SetField(mt, "__le", L.NewFunction(func(L *lua.LState) int {
		return performDecimalCmpOp(L, mt, decimal.Decimal.LessThanOrEqual)
	}))
	L.SetField(mt, "__pow", L.NewFunction(func(L *lua.LState) int {
		return performDecimalOp(L, mt, decimal.Decimal.Pow)
	}))
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"add": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Add)
		},
		"sub": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Sub)
		},
		"mul": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Mul)
		},
		"div": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Div)
		},
		"neg": func(L *lua.LState) int {
			dec := checkDecimal(1, L).Neg()
			ud := L.NewUserData()
			ud.Value = &dec
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		},
		"mod": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Mod)
		},
		"pow": func(L *lua.LState) int {
			return performDecimalOp(L, mt, decimal.Decimal.Pow)
		},
		"eq": func(L *lua.LState) int {
			return performDecimalCmpOp(L, mt, decimal.Decimal.Equal)
		},
		"lt": func(L *lua.LState) int {
			return performDecimalCmpOp(L, mt, decimal.Decimal.LessThan)
		},
		"le": func(L *lua.LState) int {
			return performDecimalCmpOp(L, mt, decimal.Decimal.LessThanOrEqual)
		},
		"gt": func(L *lua.LState) int {
			return performDecimalCmpOp(L, mt, decimal.Decimal.GreaterThan)
		},
		"ge": func(L *lua.LState) int {
			return performDecimalCmpOp(L, mt, decimal.Decimal.GreaterThanOrEqual)
		},
		"tan": func(L *lua.LState) int {
			return performDecimalMathOp(L, mt, decimal.Decimal.Tan)
		},
		"cos": func(L *lua.LState) int {
			return performDecimalMathOp(L, mt, decimal.Decimal.Cos)
		},
		"sin": func(L *lua.LState) int {
			return performDecimalMathOp(L, mt, decimal.Decimal.Sin)
		},
		"sqrt": func(L *lua.LState) int {
			if argc := L.GetTop(); argc == 1 {
				a := checkDecimal(1, L)
				res := a.Pow(constHalf)
				ud := L.NewUserData()
				ud.Value = &res
				L.SetMetatable(ud, mt)
				L.Push(ud)
				return 1
			}
			L.Push(lua.LNil)
			return 1
		},
		"log": func(L *lua.LState) int {
			original := *checkDecimal(1, L)
			var dec decimal.Decimal
			switch argc := L.GetTop(); argc {
			case 3:
				var err error
				precision := int32(L.CheckInt64(3))
				if dec, err = original.Ln(precision); err != nil {
					L.ArgError(1, err.Error())
				}
				base := decimal.NewFromInt(L.CheckInt64(2))
				if base, err = base.Ln(precision); err != nil {
					L.ArgError(2, err.Error())
				}
				dec = dec.Div(base)
			case 2:
				var err error
				if dec, err = original.Ln(32); err != nil {
					L.ArgError(1, err.Error())
				}
				base := decimal.NewFromInt(L.CheckInt64(2))
				if base, err = base.Ln(32); err != nil {
					L.ArgError(2, err.Error())
				}
				dec = dec.Div(base)
			case 1:
				fallthrough
			default:
				var err error
				if dec, err = original.Ln(32); err != nil {
					L.ArgError(1, err.Error())
				}
			}
			ud := L.NewUserData()
			ud.Value = &dec
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		},
		"abs": func(L *lua.LState) int {
			return performDecimalMathOp(L, mt, decimal.Decimal.Abs)
		},
		"round": func(L *lua.LState) int {
			return performDecimalRoundingOp(L, mt, decimal.Decimal.Round)
		},
		"floor": func(L *lua.LState) int {
			switch argc := L.GetTop(); argc {
			case 1:
				res := checkDecimal(1, L).Floor()
				ud := L.NewUserData()
				ud.Value = &res
				L.SetMetatable(ud, mt)
				L.Push(ud)
				return 1
			case 2:
				return performDecimalRoundingOp(L, mt, decimal.Decimal.RoundFloor)
			default:
				L.Push(lua.LNil)
				return 1
			}
		},
		"ceil": func(L *lua.LState) int {
			switch argc := L.GetTop(); argc {
			case 1:
				res := checkDecimal(1, L).Ceil()
				ud := L.NewUserData()
				ud.Value = &res
				L.SetMetatable(ud, mt)
				L.Push(ud)
				return 1
			case 2:
				return performDecimalRoundingOp(L, mt, decimal.Decimal.RoundCeil)
			default:
				L.Push(lua.LNil)
				return 1
			}
		},
		"string": func(L *lua.LState) int {
			L.Push(lua.LString(checkDecimal(1, L).String()))
			return 1
		},
		"float": func(L *lua.LState) int {
			if argc := L.GetTop(); argc == 1 {
				before := checkDecimal(1, L)
				after, exact := before.Float64()
				if !exact {
					zap.L().Warn(
						"precision lost after conversion",
						zap.String("before", before.String()),
						zap.Float64("after", after),
					)
				}
				L.Push(lua.LNumber(after))
				return 1
			}
			L.Push(lua.LNil)
			return 1
		},
		"isInteger": func(L *lua.LState) int {
			L.Push(lua.LBool(checkDecimal(1, L).IsInteger()))
			return 1
		},
		"isPositive": func(L *lua.LState) int {
			L.Push(lua.LBool(checkDecimal(1, L).IsPositive()))
			return 1
		},
		"isNegative": func(L *lua.LState) int {
			L.Push(lua.LBool(checkDecimal(1, L).IsNegative()))
			return 1
		},
		"isZero": func(L *lua.LState) int {
			L.Push(lua.LBool(checkDecimal(1, L).IsZero()))
			return 1
		},
	}))
	return []TypeDescriptor{
		&decimalDescriptor{},
	}
}

func init() {
	Register(RegisterDecimalType)
}
