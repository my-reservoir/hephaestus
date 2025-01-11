package biz

import (
	"context"
	er "errors"
	"github.com/google/uuid"
	"github.com/google/wire"
	"hephaestus/internal/conf"
	"hephaestus/internal/lua"
	"strings"
)

var (
	ProviderSet = wire.NewSet(
		NewLuaManager,
	)
	ErrMultiplePairsFound = er.New("multiple pairs found")
)

type KVStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	HasKeyPrefix(prefix string) (key string, exist bool)
	KeysWithPrefix(prefix string) (keys []string)
}

type LuaManager struct {
	kv KVStore
}

func NewLuaManager(store KVStore, registry *conf.Registry) *LuaManager {
	lua.NewRegistryDiscovery(registry)
	return &LuaManager{kv: store}
}

func (m *LuaManager) NewKey(ctx context.Context) (str string, err error) {
	var uid uuid.UUID
	uid, err = uuid.NewUUID()
	if err != nil {
		for {
			select {
			default:
				if uid, err = uuid.NewUUID(); err == nil {
					break
				}
			case <-ctx.Done():
				return "", err
			}
		}
	}
	str = strings.ReplaceAll(uid.String(), "-", "")
	cntNewedKeys.Inc()
	return
}

func (m *LuaManager) Set(key, script string) error {
	compiled, err := lua.CompileString(script)
	cntCompiledScripts.Inc()
	if err != nil {
		cntFailedCompiledScripts.Inc()
		return err
	}
	return m.kv.Set(key, compiled)
}

func (m *LuaManager) Exists(prefix string) (string, bool) {
	return m.kv.HasKeyPrefix(prefix)
}

func (m *LuaManager) ScriptIdByPrefix(prefix string, limit int) []string {
	keys := m.kv.KeysWithPrefix(prefix)
	if limit > len(keys) {
		limit = len(keys)
	}
	return keys[:limit]
}

func (m *LuaManager) Remove(key string) error {
	return m.kv.Delete(key)
}

func (m *LuaManager) Execute(key string, args ...interface{}) ([]interface{}, error) {
	byteCode, err := m.kv.Get(key)
	if err != nil {
		return nil, err
	}
	return lua.RunBytecodeFromByteArray(byteCode, args...)
}
