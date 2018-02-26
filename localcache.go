package localcache

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrTypeMismatch is a err indicate GetXXX func is not match with the real value with key.
	ErrTypeMismatch = errors.New("err: type mismatch")
	// ErrNoSuchKey indicate key not exist.
	ErrNoSuchKey = errors.New("err: no such key")
	// ErrExpiredKey indicate key has expired, but has't remove from cache.
	ErrExpiredKey = errors.New("err: expired key")
	// ErrDuplicateEvictedFunc will panic.
	ErrDuplicateEvictedFunc = errors.New("err: re-set evicted function")
	// ErrDuplicateKey indicate the key has already exist in cache.
	ErrDuplicateKey = errors.New("err: duplicate key")
)

const (
	// ExpireDuration indicate key has already expired, so set to -1.
	ExpireDuration = time.Duration(-1)

	defaultExpiration = time.Second * time.Duration(600)
)

// Entry is a container present data with expire info.
type Entry struct {
	value  interface{}
	expire time.Time
}

// CacheStat store cache stats.
type CacheStat struct {
	Entries int64
	Expired int64
}

type CacheConfig struct {
	Expiration time.Duration
}

func NewCacheConfig() *CacheConfig {
	return &CacheConfig{
		Expiration: defaultExpiration,
	}
}

// LocalCache is an in-memory struct store key-value pairs.
type LocalCache struct {
	data       map[interface{}]*Entry
	mu         sync.RWMutex
	expiration time.Duration
	evicted    func(key interface{}, value *Entry)
	stats      *CacheStat
}

// ResponseEntry is a wrapper of response data.
type ResponseEntry struct {
	Valid bool
	Value interface{}
}

var nilResponse = &ResponseEntry{false, nil}

// NewLocalCache return a empty LocalCache.
func NewLocalCache(configs ...*CacheConfig) *LocalCache {
	var config *CacheConfig
	if len(configs) >= 1 {
		config = configs[len(configs)-1]
	} else {
		config = NewCacheConfig()
	}
	return &LocalCache{
		data:       make(map[interface{}]*Entry),
		expiration: config.Expiration,
		stats:      &CacheStat{0, 0},
	}
}

// SetEvictedFunc set evicted func, this must be called no more once.
func (c *LocalCache) SetEvictedFunc(f func(interface{}, *Entry)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evicted != nil {
		panic(ErrDuplicateEvictedFunc)
	}
	c.evicted = f
}

// Add will do same as Set but return an error if key exists.
func (c *LocalCache) Add(key interface{}, value interface{}) error {
	return c.AddWithExpire(key, value, c.expiration)
}

// AddWithExpire will do same as SetWithExpire but return an error if key exists.
func (c *LocalCache) AddWithExpire(key interface{}, value interface{}, duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = make(map[interface{}]*Entry)
	}
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			return ErrDuplicateKey
		}
	}
	if duration <= 0 {
		c.data[key] = &Entry{value: value, expire: time.Time{}}
	} else {
		c.data[key] = &Entry{value: value, expire: time.Now().Add(duration)}
	}
	c.stats.Entries++
	return nil
}

// Set set key-value with default expiration.
func (c *LocalCache) Set(key string, value interface{}) {
	c.SetWithExpire(key, value, c.expiration)
}

// SetWithExpire set key-value with user setup expiration.
func (c *LocalCache) SetWithExpire(key interface{}, value interface{}, duration time.Duration) {
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[interface{}]*Entry)
	}
	e := &Entry{
		value: value,
	}
	if duration <= 0 {
		e.expire = time.Time{}
	} else {
		e.expire = time.Now().Add(duration)
	}
	c.data[key] = e
	c.stats.Entries++
	c.mu.Unlock()
}

// Get get the value associated by a key or an error.
func (c *LocalCache) Get(key interface{}) (v interface{}, err error) {
	v, _, err = c.GetWithExpire(key)
	return
}

// GetWithExpire get the value and left life associated by a key or an error.
func (c *LocalCache) GetWithExpire(key interface{}) (v interface{}, expire time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			return e.value, e.expire.Sub(time.Now()), nil
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
		return nil, ExpireDuration, ErrExpiredKey
	}
	return nil, ExpireDuration, ErrNoSuchKey
}

// GetEntry get a response entry which explain usability of the value or an error.
func (c *LocalCache) GetEntry(key interface{}) (v *ResponseEntry, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			return &ResponseEntry{true, e.value}, nil
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
		return nilResponse, ErrExpiredKey
	}
	return nilResponse, ErrNoSuchKey
}

// GetKeysEntry get a map of Key-ResponseEntry which explain usability of the value.
func (c *LocalCache) GetKeysEntry(keys []interface{}) (v map[interface{}]*ResponseEntry) {
	v = make(map[interface{}]*ResponseEntry)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		if e, ok := c.data[key]; ok {
			if e.expire.IsZero() || e.expire.After(time.Now()) {
				v[key] = &ResponseEntry{
					Valid: true,
					Value: e.value,
				}
			} else {
				c.stats.Entries--
				c.stats.Expired++
				if c.evicted != nil {
					c.evicted(key, e)
				}
				delete(c.data, key)
				v[key] = nilResponse
			}
		} else {
			v[key] = nilResponse
		}
	}
	return
}

// GetBool get bool value associated by key or an error.
func (c *LocalCache) GetBool(key interface{}) (v bool, err error) {
	e, err := c.Get(key)
	if err != nil {
		return false, err
	}
	switch e.(type) {
	default:
	case bool:
		return e.(bool), nil
	}
	return false, ErrTypeMismatch
}

// GetInt64 get int64 value associated by key or an error.
func (c *LocalCache) GetInt64(key interface{}) (v int64, err error) {
	e, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	switch e.(type) {
	default:
	case int:
		return int64(e.(int)), nil
	case int8:
		return int64(e.(int8)), nil
	case int16:
		return int64(e.(int16)), nil
	case int32:
		return int64(e.(int32)), nil
	case int64:
		return int64(e.(int64)), nil
	}
	return 0, ErrTypeMismatch
}

// GetUint64 get uint64 value associated by key or an error.
func (c *LocalCache) GetUint64(key interface{}) (v uint64, err error) {
	e, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	switch e.(type) {
	default:
	case uint:
		return uint64(e.(uint)), nil
	case uint8:
		return uint64(e.(uint8)), nil
	case uint16:
		return uint64(e.(uint16)), nil
	case uint32:
		return uint64(e.(uint32)), nil
	case uint64:
		return uint64(e.(uint64)), nil
	}
	return 0, ErrTypeMismatch
}

// GetFloat64 get float64 value associated by key or an error.
func (c *LocalCache) GetFloat64(key interface{}) (v float64, err error) {
	e, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	switch e.(type) {
	default:
	case float32:
		return float64(e.(float64)), nil
	case float64:
		return e.(float64), nil
	}
	return 0, ErrTypeMismatch
}

// GetString get string value associated by key or an error.
func (c *LocalCache) GetString(key interface{}) (v string, err error) {
	e, err := c.Get(key)
	if err != nil {
		return "", err
	}
	switch e.(type) {
	default:
	case string:
		return e.(string), nil
	case []byte:
		return string(e.([]byte)), nil
	}
	return "", ErrTypeMismatch
}

// GetByte get byte value associated by key or an error.
func (c *LocalCache) GetByte(key interface{}) (v byte, err error) {
	e, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	switch e.(type) {
	default:
	case byte:
		return e.(byte), nil
	case int8:
		return byte(e.(int8)), nil
	}
	return 0, ErrTypeMismatch
}

// GetRune get rune value associated by key or an error.
func (c *LocalCache) GetRune(key interface{}) (v rune, err error) {
	e, err := c.Get(key)
	if err != nil {
		return 0, err
	}
	switch e.(type) {
	default:
	case rune:
		return e.(rune), nil
	}
	return 0, ErrTypeMismatch
}

// Expire to expire a key immediately, ignore the default and left expiration.
func (c *LocalCache) Expire(key interface{}) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
	}
	return
}

// Flush will reset all data in cache, but stats will be keeped.
func (c *LocalCache) Flush() {
	c.mu.Lock()
	if c.evicted != nil {
		for k, e := range c.data {
			c.evicted(k, e)
		}
	}
	c.data = make(map[interface{}]*Entry)
	c.stats.Expired += c.stats.Entries
	c.stats.Entries = 0
	c.mu.Unlock()
}

// Reset will reset both data and stats.
func (c *LocalCache) Reset() {
	c.mu.Lock()
	if c.evicted != nil {
		for k, e := range c.data {
			c.evicted(k, e)
		}
	}
	c.data = make(map[interface{}]*Entry)
	c.stats = &CacheStat{0, 0}
	c.mu.Unlock()
}

// Stats return cache stats.
func (c *LocalCache) Stats() *CacheStat {
	c.mu.Lock()
	stats := c.stats
	c.mu.Unlock()
	return stats
}
