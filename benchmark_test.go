package localcache_test

import (
	"sync"
	"testing"

	"github.com/leaxoy/localcache"
)

type entry struct {
	value interface{}
	valid bool
}

func BenchmarkStdMapIntInterface(b *testing.B) {
	var mu sync.Mutex
	m := make(map[int]interface{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[i] = i
		mu.Unlock()
	}
}

func BenchmarkStdMapStringString(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]string)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
	}
}

func BenchmarkStdMapStringStringSetDelete(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]string)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		delete(m, "foo")
		mu.Unlock()
	}
}

func BenchmarkStdMapStringInterface(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]interface{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = i
		mu.Unlock()
	}
}

func BenchmarkStdMapStringStruct(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]entry)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = entry{
			valid: false,
			value: i,
		}
		mu.Unlock()
	}
}

func BenchmarkStdMapStringStringStruct(b *testing.B) {
	var mu sync.Mutex
	m := make(map[string]entry)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = entry{
			valid: false,
			value: "bar",
		}
		mu.Unlock()
	}
}

func BenchmarkStdMapInterfaceInterface(b *testing.B) {
	var mu sync.Mutex
	m := make(map[interface{}]interface{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[i] = i
		mu.Unlock()
	}
}

func BenchmarkStdMapInterfaceStructSet(b *testing.B) {
	var mu sync.Mutex
	m := make(map[interface{}]entry)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m[i] = entry{
			value: 1,
			valid: false,
		}
		mu.Unlock()
	}
}

func BenchmarkLocalCache_Add(b *testing.B) {
	localCache := localcache.NewLocalCache(nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		localCache.Add("bar", i)
	}
}

func BenchmarkLocalCache_Set(b *testing.B) {
	localCache := localcache.NewLocalCache(nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		localCache.Set("bar", i)
	}
}
