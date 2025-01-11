package lua

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
	grpc2 "google.golang.org/grpc"
	"io"
	"reflect"
	"strings"
	"time"
)

type MethodIdentifier struct {
	method string
	path   *string
}

type Client interface {
	Invoke(ctx context.Context, method *MethodIdentifier, args interface{}, reply interface{}) error
	io.Closer
}

type ClientType uint8

const (
	HTTP ClientType = iota
	GRPC
)

const (
	HttpGet    = "GET"
	HttpPost   = "POST"
	HttpPut    = "PUT"
	HttpDelete = "DELETE"
)

type httpClient struct {
	conn *http.Client
}

func Get(path string) (i *MethodIdentifier) {
	return &MethodIdentifier{
		method: HttpGet,
		path:   &path,
	}
}

func Post(path string) (i *MethodIdentifier) {
	return &MethodIdentifier{
		method: HttpPost,
		path:   &path,
	}
}

func Put(path string) (i *MethodIdentifier) {
	return &MethodIdentifier{
		method: HttpPut,
		path:   &path,
	}
}

func Delete(path string) (i *MethodIdentifier) {
	return &MethodIdentifier{
		method: HttpDelete,
		path:   &path,
	}
}

func RpcMethod(method string) (i *MethodIdentifier) {
	return &MethodIdentifier{
		method: method,
		path:   nil,
	}
}

func (c *httpClient) Close() error {
	return c.conn.Close()
}

func (c *httpClient) Invoke(ctx context.Context, method *MethodIdentifier, args interface{}, reply interface{}) error {
	return c.conn.Invoke(ctx, method.method, *method.path, args, reply)
}

type grpcClient struct {
	conn *grpc2.ClientConn
}

func (c *grpcClient) Close() error {
	return c.conn.Close()
}

func (c *grpcClient) Invoke(ctx context.Context, method *MethodIdentifier, args interface{}, reply interface{}) error {
	return c.conn.Invoke(ctx, method.method, args, reply)
}

var reg registry.Discovery

func client(endpoint string, clientType ClientType) (Client, error) {
	if c, ok := clients[endpoint]; ok {
		return c[clientType], nil
	}
	if !strings.HasPrefix(endpoint, "discovery://") {
		endpoint = "discovery://" + endpoint
	}
	switch clientType {
	default:
		fallthrough
	case HTTP:
		conn, err := http.NewClient(
			context.Background(),
			http.WithEndpoint(endpoint),
			http.WithDiscovery(reg),
			http.WithBlock(),
		)
		if err != nil {
			return nil, err
		}
		return &httpClient{conn: conn}, nil
	case GRPC:
		conn, err := grpc.Dial(
			context.Background(),
			grpc.WithEndpoint(endpoint),
			grpc.WithDiscovery(reg),
		)
		if err != nil {
			return nil, err
		}
		return &grpcClient{conn: conn}, nil
	}
}

var clients = make(map[string]map[ClientType]Client)

func luaType(value interface{}) lua.LValue {
	switch value.(type) {
	case float32, float64,
		int, uint, uint8, uint16, uint32, uint64, int8, int16, int32, int64:
		return lua.LNumber(value.(float64))
	case string:
		return lua.LString(value.(string))
	case bool:
		return lua.LBool(value.(bool))
	case nil:
		return lua.LNil
	}
	return lua.LNil
}

func handleHttpRequest(L *lua.LState, method string) int {
	switch argc := L.GetTop(); {
	case argc >= 2:
		c, uri := L.CheckUserData(1).Value, L.CheckString(2)
		v, ok := c.(Client)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var args interface{}
		if argc >= 3 {
			arguments := make(map[string]interface{})
			L.CheckTable(3).ForEach(func(key lua.LValue, value lua.LValue) {
				var val interface{}
				switch value.Type() {
				case lua.LTNumber:
					val = float64(lua.LVAsNumber(value))
				case lua.LTString:
					val = lua.LVAsString(value)
				case lua.LTBool:
					val = lua.LVAsBool(value)
				default:
				}
				arguments[lua.LVAsString(key)] = val
			})
			args = arguments
		}
		var reply map[string]interface{}
		if err := v.Invoke(ctx, &MethodIdentifier{method: method, path: &uri}, args, &reply); err == nil {
			ud := L.NewTable()
			for k, v := range reply {
				ud.RawSetString(k, luaType(v))
			}
			L.Push(ud)
			return 1
		} else {
			fmt.Println(reflect.TypeOf(reply))
			zap.L().Warn("failed to invoke remote method", zap.Error(err))
			L.Push(lua.LNil)
			return 1
		}
	case argc == 1:
		fallthrough
	default:
		L.ArgError(1, fmt.Sprintf("not enough arguments, at least 1 but %d is provided", argc-1))
		L.Push(lua.LNil)
		return 1
	}
}

type service struct {
	endpoint string
}

func checkService(n int, L *lua.LState) *service {
	if L.CheckAny(n).Type() == lua.LTUserData {
		if v, ok := L.CheckUserData(n).Value.(*service); ok {
			return v
		}
		return nil
	}
	return nil
}

func RegisterServiceTypes(L *lua.LState) []TypeDescriptor {
	mt := L.NewTypeMetatable("service")
	httpMt, grpcMt := L.NewTypeMetatable("<http>"), L.NewTypeMetatable("<grpc>")
	L.SetField(httpMt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			return handleHttpRequest(L, HttpGet)
		},
		"post": func(L *lua.LState) int {
			return handleHttpRequest(L, HttpPost)
		},
		"put": func(L *lua.LState) int {
			return handleHttpRequest(L, HttpPut)
		},
		"delete": func(L *lua.LState) int {
			return handleHttpRequest(L, HttpDelete)
		},
	}))
	L.SetField(httpMt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		v := L.CheckAny(1)
		L.Push(lua.LString(fmt.Sprintf("<http service instance at %p>", &v)))
		return 1
	}))
	L.SetField(grpcMt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"invoke": func(L *lua.LState) int {
			switch argc := L.GetTop(); argc {
			default:
				L.ArgError(1, fmt.Sprintf("not enough arguments, at least 1 but %d is provided", argc-1))
				L.Push(lua.LNil)
				return 1
			}
		},
	}))
	L.SetField(grpcMt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		v := L.CheckAny(1)
		L.Push(lua.LString(fmt.Sprintf("<grpc service instance at %p>", &v)))
		return 1
	}))
	L.SetGlobal("service", mt)
	L.SetField(mt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		srv := checkService(1, L)
		if srv == nil {
			L.ArgError(1, "unexpected type")
			L.Push(lua.LNil)
			return 1
		}
		L.Push(lua.LString(fmt.Sprintf("<service at %p>: {endpoint: %s}", srv, srv.endpoint)))
		return 1
	}))
	L.SetField(mt, "discover", L.NewFunction(func(L *lua.LState) int {
		if argc := L.GetTop(); argc >= 1 {
			endpoint := L.CheckString(1)
			if len(endpoint) == 0 {
				L.ArgError(1, "endpoint should not be empty")
				L.Push(lua.LNil)
				return 1
			}
			ud := L.NewUserData()
			ud.Value = &service{endpoint: endpoint}
			L.SetMetatable(ud, mt)
			L.Push(ud)
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"http": func(L *lua.LState) int {
			srv := checkService(1, L)
			if srv == nil {
				L.ArgError(1, "unexpected type")
				L.Push(lua.LNil)
				return 1
			}
			if c, err := client(srv.endpoint, HTTP); err == nil {
				ud := L.NewUserData()
				ud.Value = c
				L.SetMetatable(ud, httpMt)
				L.Push(ud)
			} else {
				zap.L().Warn("failed to open HTTP client", zap.Error(err))
				L.Push(lua.LNil)
			}
			return 1
		},
		"grpc": func(L *lua.LState) int {
			srv := checkService(1, L)
			if srv == nil {
				L.ArgError(1, "unexpected type")
				L.Push(lua.LNil)
				return 1
			}
			if c, err := client(srv.endpoint, GRPC); err == nil {
				ud := L.NewUserData()
				ud.Value = c
				L.SetMetatable(ud, grpcMt)
				L.Push(ud)
			} else {
				zap.L().Warn("failed to open gRPC client", zap.Error(err))
				L.Push(lua.LNil)
			}
			return 1
		},
		"endpoint": func(L *lua.LState) int {
			srv := checkService(1, L)
			if srv == nil {
				L.ArgError(1, "unexpected type")
				L.Push(lua.LNil)
				return 1
			}
			L.Push(lua.LString(srv.endpoint))
			return 1
		},
	}))
	return []TypeDescriptor{}
}

func init() {
	Register(RegisterServiceTypes)
}
