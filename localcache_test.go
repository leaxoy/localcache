package localcache_test

import (
	"dokigo/localcache"
	"log"
	"reflect"
	"testing"
	"time"
)

var testCases = map[string]interface{}{
	"bool":    false,
	"int":     -1,
	"int8":    int8(-8),
	"int16":   int(-16),
	"int32":   int32(-32),
	"int64":   int64(-64),
	"uint":    uint(1),
	"uint8":   uint8(8),
	"uint16":  uint16(16),
	"uint32":  uint32(32),
	"uint64":  uint64(64),
	"float64": 0.1234,
	"float32": float32(0.123),
	"string":  "abcd",
	"bytes":   []byte("abc"),
	"byte":    'a',
	"rune":    '漓',
}

var evictedFunc = func(key string, entry *localcache.Entry) {
	log.Printf("dump entry, key: %s, entry: %+v\n", key, entry)
}

func TestGet(t *testing.T) {
	var localCache = localcache.NewLocalCache(3600)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	for key, value := range testCases {
		localCache.Set(key, value)
	}
	for key, value := range testCases {
		v, err := localCache.Get(key)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(v, value) {
			t.Errorf("err: not equal, expect: %+v, but got: %+v\n", value, v)
		}
	}
}

func TestGetExpiration(t *testing.T) {
	var localCache = localcache.NewLocalCache(3600)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	localCache.Set("long", 123)
	localCache.SetWithExpire("short", 1, 1)
	v, err := localCache.Get("long")
	if err != nil {
		t.Error(err)
		goto shortValid
	}
	if !reflect.DeepEqual(v, 123) {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 123, v)
	}
shortValid:
	v, err = localCache.Get("short")
	if err != nil {
		t.Error(err)
		goto longInvalid
	}
	if !reflect.DeepEqual(v, 1) {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 1, v)
	}
longInvalid:
	time.Sleep(time.Second)
	v, err = localCache.Get("long")
	if err != nil {
		t.Error(err)
		goto shortInvalid
	}
	if !reflect.DeepEqual(v, 123) {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 123, v)
	}
shortInvalid:
	v, err = localCache.Get("short")
	if err != localcache.ErrKeyExpired {
		t.Fatal(err)
	}
}

func TestGetBool(t *testing.T) {
	var localCache = localcache.NewLocalCache(3600)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	localCache.Set("f", false)
	localCache.Set("t", true)
	v, err := localCache.GetBool("f")
	if err != nil {
		t.Error(err)
	}
	if v {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", false, v)
	}
	v, err = localCache.GetBool("f")
	if err != nil {
		t.Error(err)
	}
	if v {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", false, v)
	}
}

func TestGetByte(t *testing.T) {
	var localCache = localcache.NewLocalCache(3600)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	localCache.Set("xxx", byte('b'))
	v, err := localCache.GetByte("xxx")
	if err != nil {
		t.Error(err)
		return
	}
	if v != 'b' {
		t.Errorf("err: not equal, expect: %+v, but got %+v\n", 'b', v)
	}
}

func TestExpire(t *testing.T) {
	var localCache = localcache.NewLocalCache(3600)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	localCache.Set("xxx", 1234)
	v, err := localCache.GetInt64("xxx")
	if err != nil {
		t.Error(err)
	}
	if v != 1234 {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 1234, v)
	}
	err = localCache.Expire("xxx")
	if err != nil {
		t.Error(err)
	}
	_, err = localCache.Get("xxx")
	if err != localcache.ErrNoSuchKey {
		t.Error(err)
	}
}

func TestTimeoutExpiration(t *testing.T) {
	var localCache = localcache.NewLocalCache(1)
	localCache.SetEvictedFunc(evictedFunc)
	defer localCache.Flush()
	localCache.Set("xxx", 1234)
	v, err := localCache.GetInt64("xxx")
	if err != nil {
		t.Error(err)
	}
	if v != 1234 {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 1234, v)
	}
	time.Sleep(time.Second)
	_, err = localCache.GetInt64("xxx")
	if err != localcache.ErrKeyExpired {
		t.Error(err)
	}
}
