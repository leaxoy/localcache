package localcache_test

import (
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/leaxoy/localcache"
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

var evictedFunc = func(key localcache.Key, entry localcache.Entry) {
	log.Printf("dump entry, key: %s, entry: %+v\n", key, entry)
}

func TestGet(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	defer localCache.Flush()
	for key, value := range testCases {
		localCache.Set(localcache.Key(key), value)
	}
	for key, value := range testCases {
		v, err := localCache.Get(localcache.Key(key))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(v, value) {
			t.Errorf("err: not equal, expect: %+v, but got: %+v\n", value, v)
		}
	}
}

func TestGetExpiration(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	defer localCache.Flush()
	localCache.Set("long", 123)
	localCache.SetWithExpire("short", 1, time.Second)
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
	if err != localcache.ErrExpiredKey {
		t.Fatal(err)
	}
}

func TestExpire(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
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
	var localCache = localcache.NewLocalCache(&localcache.CacheConfig{Expiration: time.Second})
	localCache.Set("xxx", 1234)
	v, err := localCache.GetInt64("xxx")
	if err != nil {
		t.Error(err)
		return
	}
	if v != 1234 {
		t.Errorf("err: not equal, expect: %+v, but got: %+v\n", 1234, v)
	}
	time.Sleep(time.Second)
	_, err = localCache.GetInt64("xxx")
	if err != localcache.ErrExpiredKey {
		t.Error(err)
	}
}

func TestLocalCache_SetEvictedFunc(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	defer func() {
		if e := recover(); e != localcache.ErrDuplicateEvictedFunc {
			t.Error(e)
		}
	}()
	localCache.SetEvictedFunc(evictedFunc)
	localCache.SetEvictedFunc(evictedFunc)
}

func TestLocalCache_GetString(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.Set("1", "123")
	v, err := localCache.GetString("1")
	if err != nil {
		t.Error(err)
	}
	if v != "123" {
		t.Errorf("err: not equal, expect %+v, but got %+v\n", "123", v)
	}
}

func TestLocalCache_GetByte(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
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

func TestLocalCache_GetUint64(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.Add("123", uint8(9))
	v, err := localCache.GetUint64("123")
	if err != nil {
		t.Error(err)
		return
	}
	if v != uint64(9) {
		t.Errorf("err: not equal, expect: %+v, but got %+v\n", 'b', v)
	}
}

func TestLocalCache_GetBool(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
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

func TestLocalCache_Add(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.Add("123", 456)
	v, err := localCache.GetInt64("123")
	if err != nil {
		t.Error(err)
		return
	}
	if v != 456 {
		t.Errorf("err: not euqal, expect: %+v, but got: %+v\n", 456, v)
	}
}

func TestLocalCache_AddWithExpire(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.AddWithExpire("123", 456, time.Second)
	v, err := localCache.GetInt64("123")
	if err != nil {
		t.Error(err)
		return
	}
	if v != 456 {
		t.Errorf("err: not euqal, expect: %+v, but got: %+v\n", 456, v)
	}
	time.Sleep(time.Second)
	v, err = localCache.GetInt64("123")
	if err != localcache.ErrExpiredKey {
		t.Error(err)
		return
	}
	if v != 0 {
		t.Errorf("err: not euqal, expect: %+v, but got: %+v\n", 0, v)
	}
}

func TestLocalCache_Reset(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.Add("123", 456)
	localCache.Reset()
}

func TestLocalCache_Stats(t *testing.T) {
	var stats *localcache.CacheStat
	var localCache = localcache.NewLocalCache(nil)
	localCache.Add("123", 456)
	stats = localCache.Stats()
	if stats.Entries != 1 || stats.Expired != 0 {
		t.Errorf("err: not equal, extires expect %+v, but got %+v, expires expect +%v, but got %+v\n", 1, stats.Entries, 0, stats.Expired)
	}
	localCache.AddWithExpire("456", 123, time.Second)
	stats = localCache.Stats()
	if stats.Entries != 2 || stats.Expired != 0 {
		t.Errorf("err: not equal, extires expect %+v, but got %+v, expires expect +%v, but got %+v\n", 2, stats.Entries, 0, stats.Expired)
	}
}

func TestLocalCache_GetEntry(t *testing.T) {
	var localCache = localcache.NewLocalCache(nil)
	localCache.Add("xxx", false)
	entry, err := localCache.GetEntry("xxx")
	if err != nil {
		t.Error(err)
	}
	if !entry.Valid {
		t.Errorf("err: not valid, expect valid, but got invalid")
	}
	switch entry.Value.(type) {
	case bool:
		if entry.Value.(bool) {
			t.Errorf("err: not equal, expect false, but got true")
		}
	default:
		t.Errorf("err: typemismatch, expect bool, but got %s", reflect.TypeOf(entry.Value))
	}
}

func TestLocalCache_GetKeysEntry(t *testing.T) {

}
