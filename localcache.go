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
	// ErrKeyExpired indicate key has expired, but has't remove from cache.
	ErrKeyExpired = errors.New("err: key already expired")
	// ErrDuplicateEvictedFunc will panic.
	ErrDuplicateEvictedFunc = errors.New("err: re-set evicted function")
)

const (
	// ExpireDuration indicate key has already expired, so set to -1.
	ExpireDuration = time.Duration(-1)
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

// LocalCache is an in-memory struct store key-value pairs.
type LocalCache struct {
	data       map[string]*Entry
	mu         sync.RWMutex
	expiration time.Duration
	evicted    func(key string, value *Entry)
	stats      *CacheStat
}

// ResponseEntry is a wrapper of response data.
type ResponseEntry struct {
	Valid bool
	Value interface{}
}

var nilResponse = &ResponseEntry{false, nil}

// NewLocalCache return a empty LocalCache.
func NewLocalCache(defaultExpiration int64) *LocalCache {
	return &LocalCache{
		data:       make(map[string]*Entry),
		expiration: time.Second * time.Duration(defaultExpiration),
		stats:      &CacheStat{0, 0},
	}
}

// SetEvictedFunc set evicted func, this must be called no more once.
func (c *LocalCache) SetEvictedFunc(f func(string, *Entry)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evicted != nil {
		panic(ErrDuplicateEvictedFunc)
	}
	c.evicted = f
}

// Set set key-value with default expiration.
func (c *LocalCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data == nil {
		c.data = make(map[string]*Entry)
	}
	c.data[key] = &Entry{
		value:  value,
		expire: time.Now().Add(c.expiration),
	}
	c.stats.Entries++
}

// SetWithExpire set key-value with user setup expiration.
func (c *LocalCache) SetWithExpire(key string, value interface{}, duration int64) {
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[string]*Entry)
	}
	e := &Entry{
		value: value,
	}
	if duration <= 0 {
		e.expire = time.Time{}
	} else {
		e.expire = time.Now().Add(time.Duration(duration) * time.Second)
	}
	c.data[key] = e
	c.stats.Entries++
	c.mu.Unlock()
}

// Get get the value associated by a key or an error.
func (c *LocalCache) Get(key string) (v interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			return e.value, nil
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
		return nil, ErrKeyExpired
	}
	return nil, ErrNoSuchKey
}

// GetWithExpire get the value and left life associated by a key or an error.
func (c *LocalCache) GetWithExpire(key string) (v interface{}, expire time.Duration, err error) {
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
		return nil, ExpireDuration, ErrKeyExpired
	}
	return nil, ExpireDuration, ErrNoSuchKey
}

// GetEntry get a response entry which explain usability of the value or an error.
func (c *LocalCache) GetEntry(key string) (v *ResponseEntry, err error) {
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
		return nilResponse, ErrKeyExpired
	}
	return nilResponse, ErrNoSuchKey
}

// GetKeysEntry get a map of Key-ResponseEntry which explain usability of the value.
func (c *LocalCache) GetKeysEntry(keys []string) (v map[string]*ResponseEntry) {
	v = make(map[string]*ResponseEntry)
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
func (c *LocalCache) GetBool(key string) (v bool, err error) {
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
func (c *LocalCache) GetInt64(key string) (v int64, err error) {
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
func (c *LocalCache) GetUint64(key string) (v uint64, err error) {
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
func (c *LocalCache) GetFloat64(key string) (v float64, err error) {
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
func (c *LocalCache) GetString(key string) (v string, err error) {
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
func (c *LocalCache) GetByte(key string) (v byte, err error) {
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
func (c *LocalCache) GetRune(key string) (v rune, err error) {
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
func (c *LocalCache) Expire(key string) (err error) {
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
	c.data = make(map[string]*Entry)
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
	c.data = make(map[string]*Entry)
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
