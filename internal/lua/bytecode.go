package lua

import (
	"bufio"
	"bytes"
	"encoding/gob"
	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
	"io"
	"unsafe"
)

// FuncProto is the identical struct of [lua.FunctionProto], except that it opens the
// field [FuncProto.StringConstants] so that the field would also be serialized & deserialized.
type FuncProto struct {
	SourceName         string
	LineDefined        int
	LastLineDefined    int
	NumUpvalues        uint8
	NumParameters      uint8
	IsVarArg           uint8
	NumUsedRegisters   uint8
	Code               []uint32
	Constants          []lua.LValue
	FunctionPrototypes []*FuncProto

	DbgSourcePositions []int
	DbgLocals          []*lua.DbgLocalInfo
	DbgCalls           []lua.DbgCall
	DbgUpvalues        []string

	StringConstants []string
}

func init() {
	gob.Register(FuncProto{})
	gob.Register(lua.LTable{})
	gob.Register(lua.LNil)
	gob.Register(lua.LFunction{})
	gob.Register(lua.LNumber(0))
	gob.Register(lua.LString(""))
	gob.Register(lua.LBool(false))
	gob.Register(lua.LUserData{})
}

const keyCompiledBytecode = "<compiled>"

func CompileString(s string) ([]byte, error) {
	return Compile(bytes.NewBufferString(s))
}

func Compile(reader io.Reader) (b []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err, _ = e.(error)
		}
	}()
	stmts, err := parse.Parse(reader, keyCompiledBytecode)
	if err != nil {
		return nil, err
	}
	proto, err := lua.Compile(stmts, keyCompiledBytecode)
	if err != nil {
		return nil, err
	}
	var code bytes.Buffer
	if err := gob.NewEncoder(&code).Encode(*(*FuncProto)(unsafe.Pointer(proto))); err != nil {
		return nil, err
	}
	return code.Bytes(), nil
}

func RunBytecodeFromByteArray(b []byte, args ...interface{}) ([]interface{}, error) {
	return RunBytecode(bytes.NewBuffer(b), args...)
}

func FunctionProtoFromBytecode(reader io.Reader) (fn *lua.FunctionProto, err error) {
	var proto FuncProto
	if err = gob.NewDecoder(bufio.NewReader(reader)).Decode(&proto); err != nil {
		return
	}
	fn = (*lua.FunctionProto)(unsafe.Pointer(&proto))
	return
}

func RunBytecode(reader io.Reader, args ...interface{}) (returns []interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err, _ = e.(error)
		}
	}()
	var proto FuncProto
	if err := gob.NewDecoder(bufio.NewReader(reader)).Decode(&proto); err != nil {
		return nil, err
	}
	vm := defaultPool.New()
	defer vm.Close()
	vm.Push(vm.NewFunctionFromProto((*lua.FunctionProto)(unsafe.Pointer(&proto))))
	storeGlobalThis(vm, &GlobalThis{Args: args})
	defer deleteGlobalThis(vm)
	if err = vm.PCall(0, lua.MultRet, nil); err != nil {
		return nil, err
	}
	returns = loadGlobalThis(vm).Ret
	return
}
