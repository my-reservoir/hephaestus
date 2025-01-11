package data

import (
	"errors"
	"github.com/cockroachdb/pebble"
	"github.com/go-kratos/kratos/v2/log"
	art "github.com/plar/go-adaptive-radix-tree"
	"go.uber.org/zap"
	"hephaestus/internal/biz"
	"os"
	"path/filepath"
	"time"
)

type kvStore struct {
	db        *pebble.DB
	writeOpts *pebble.WriteOptions
	art       art.Tree
}

func (k *kvStore) HasKeyPrefix(prefix string) (key string, exist bool) {
	if len(prefix) < 32 {
		k.art.ForEachPrefix([]byte(prefix), func(node art.Node) (cont bool) {
			key = string(node.Key())
			exist = true
			return !exist
		})
	} else {
		key = prefix
		if _, err := k.Get(key); err == nil {
			exist = true
		} else if errors.Is(err, pebble.ErrNotFound) {
			exist = false
		}
	}
	return
}

func (k *kvStore) KeysWithPrefix(prefix string) (keys []string) {
	k.art.ForEachPrefix([]byte(prefix), func(node art.Node) (cont bool) {
		keys = append(keys, string(node.Key()))
		return true
	})
	return
}

func (k *kvStore) Get(key string) ([]byte, error) {
	bytesKey := []byte(key)
	// If the key's length is less than 32, then we check whether the key presents in the tree.
	if len(bytesKey) < 32 {
		count := 0
		k.art.ForEachPrefix(bytesKey, func(node art.Node) (cont bool) {
			bytesKey = node.Key()
			count++
			return count < 2
		})
		if count >= 2 {
			return nil, biz.ErrMultiplePairsFound
		}
	}
	val, cleanup, err := k.db.Get(bytesKey)
	if err != nil {
		return nil, err
	}
	return val, cleanup.Close()
}

func (k *kvStore) Set(key string, value []byte) error {
	bytesKey := []byte(key)
	if _, closer, err := k.db.Get(bytesKey); err == nil {
		if err = closer.Close(); err != nil {
			log.Errorf("error closing the key-value pair with key: %s", key)
		}
	} else if errors.Is(err, pebble.ErrNotFound) {
		// if the key does not present, insert it into the ART
		k.art.Insert(bytesKey, struct{}{})
	}
	return k.db.Set(bytesKey, value, k.writeOpts)
}

func (k *kvStore) Delete(key string) error {
	bytesKey := []byte(key)
	if len(bytesKey) < 32 {
		count := 0
		k.art.ForEachPrefix(bytesKey, func(node art.Node) (cont bool) {
			bytesKey = node.Key()
			count++
			return count < 2
		})
		if count >= 2 {
			return biz.ErrMultiplePairsFound
		}
	}
	if _, closer, err := k.db.Get(bytesKey); err == nil {
		k.art.Delete(bytesKey) // if the key-value pair is found, then purge it out of the ART
		if err = closer.Close(); err != nil {
			log.Errorf("error closing the key-value pair with key: %s", key)
		}
	}
	return k.db.Delete(bytesKey, k.writeOpts)
}

func NewDefaultKVStore() (d biz.KVStore, err error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return NewKeyValueStore(filepath.Join(filepath.Dir(exe), "data"), &pebble.Options{})
}

func NewKeyValueStore(dirname string, opt *pebble.Options) (d biz.KVStore, err error) {
	var db *pebble.DB
	if opt == nil {
		opt = &pebble.Options{}
	}
	opt.Logger = log.NewHelper(log.GetLogger())
	db, err = pebble.Open(dirname, opt) // open the embedded database
	if err != nil {
		zap.L().Error("failed to open database", zap.Error(err))
		return nil, err
	}
	RegisterMetrics(db)
	dest := &kvStore{
		db:        db,
		writeOpts: &pebble.WriteOptions{Sync: true},
		art:       art.New(), // initialize the adaptive radix tree (ART)
	}
	d = dest
	// Here we start an iterator to get all the keys and save them into the ART
	go func() {
		begin := time.Now()
		iter, er := db.NewIter(&pebble.IterOptions{})
		if er != nil {
			log.Errorf("failed to create iterator")
		}
		defer func(iter *pebble.Iterator) {
			if err := iter.Close(); err != nil {
				log.Errorf("failed to close iterator: %v", err)
			}
		}(iter)
		count := 0
		for iter.First(); iter.Valid(); iter.Next() {
			dest.art.Insert(iter.Key(), struct{}{})
			count++
		}
		if count > 0 {
			log.Infof(
				"successfully loaded %d scripts into the router tree in %s",
				count, time.Now().Sub(begin).String(),
			)
		}
	}()
	return
}
