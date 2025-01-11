package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"google.golang.org/protobuf/types/known/emptypb"
	v1 "hephaestus/api/lua/v1"
	"hephaestus/internal/biz"
	"hephaestus/internal/lua"
)

var ProviderSet = wire.NewSet(
	NewHephaestusService,
)

type HephaestusService struct {
	v1.UnimplementedHephaestusServer
	mgr *biz.LuaManager
}

func NewHephaestusService(mgr *biz.LuaManager) *HephaestusService {
	return &HephaestusService{mgr: mgr}
}

func (s *HephaestusService) RunScriptOnce(ctx context.Context, c *v1.RunScriptOnceRequest) (retVal *v1.ScriptReturnedValues, err error) {
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		var args []interface{}
		if args, err = ConvertFromProto(c.Args); err != nil {
			log.Debugf("failed to convert args to lua values: %v", err)
			return
		}
		var ret []interface{}
		if ret, err = lua.Pool().RunString(c.Script, args...); err != nil {
			log.Debugf("failed to run script: %v", err)
			return
		}
		if retVal, err = ConvertArgsToProto(ret...); err != nil {
			log.Debugf("failed to convert return values to proto: %v", err)
		}
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			err = v1.ErrorContextTimeout("process of running script once is canceled")
			return
		}
	}
}

func (s *HephaestusService) AddScript(ctx context.Context, str *v1.ScriptContent) (id *v1.ScriptIdentifier, err error) {
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		var key string
		key, err = s.mgr.NewKey(ctx)
		if err != nil {
			return
		}
		if err = s.mgr.Set(key, str.Script); err != nil {
			return
		}
		id = &v1.ScriptIdentifier{Id: key}
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			return nil, v1.ErrorContextTimeout("process of adding script id is canceled")
		}
	}
}
func (s *HephaestusService) UpdateScript(ctx context.Context, c *v1.UpdateScriptRequest) (_ *emptypb.Empty, err error) {
	if err = c.Validate(); err != nil {
		return nil, v1.ErrorInvalidParam("failed to pass validation: %s", err.Error())
	}
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		if _, ext := s.mgr.Exists(c.Id); !ext {
			err = v1.ErrorScriptNotFound("script with id prefix %s does not exist", c.Id)
			return
		}
		err = s.mgr.Set(c.Id, c.Script)
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			return nil, v1.ErrorContextTimeout("updating process the script with id %s is canceled", c.Id)
		}
	}
}
func (s *HephaestusService) DeleteScript(ctx context.Context, id *v1.ScriptIdentifier) (_ *emptypb.Empty, err error) {
	if err = id.Validate(); err != nil {
		return nil, v1.ErrorInvalidParam("failed to pass validation: %s", err.Error())
	}
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		if _, ext := s.mgr.Exists(id.Id); !ext {
			err = v1.ErrorScriptNotFound("script with id prefix %s does not exist", id.Id)
			return
		}
		err = s.mgr.Remove(id.Id)
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			return nil, v1.ErrorContextTimeout("deletion timed out for script with id %s", id)
		}
	}
}
func (s *HephaestusService) ExecuteScript(
	ctx context.Context, req *v1.ExecuteScriptRequest,
) (retVal *v1.ScriptReturnedValues, err error) {
	if err = req.Validate(); err != nil {
		return nil, v1.ErrorInvalidParam("failed to pass validation: %s", err.Error())
	}
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		if _, ext := s.mgr.Exists(req.Id); !ext {
			err = v1.ErrorScriptNotFound("script with id prefix %s does not exist", req.Id)
			return
		}
		var args []interface{}
		if args, err = ConvertFromProto(req.Args); err != nil {
			return
		}
		var ret []interface{}
		if ret, err = s.mgr.Execute(req.Id, args...); err != nil {
			return
		}
		retVal, err = ConvertArgsToProto(ret...)
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			return nil, v1.ErrorContextTimeout("execution time for the script %s is too long", req.Id)
		}
	}
}

func (s *HephaestusService) FindScript(
	ctx context.Context, req *v1.FindScriptRequest,
) (resp *v1.ScriptIdentifiersResponse, err error) {
	ok := make(chan struct{})
	go func() {
		defer func() {
			ok <- struct{}{}
		}()
		prefix := ""
		if req.Prefix != nil {
			prefix = *req.Prefix
		}
		limit := 10
		if req.Limit != nil {
			limit = int(*req.Limit)
		}
		resp = &v1.ScriptIdentifiersResponse{Id: s.mgr.ScriptIdByPrefix(prefix, limit)}
	}()
	for {
		select {
		case <-ok:
			return
		case <-ctx.Done():
			return nil, v1.ErrorContextTimeout("query timed out")
		}
	}
}
