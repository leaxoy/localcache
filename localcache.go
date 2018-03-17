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
	defaultExpireTick = time.Minute * time.Duration(5)
)

// Key is a generic type for map key.
type Key interface{}

// Entry is a container present data with expire info.
type Entry struct {
	value  interface{}
	expire int64
}

// IsExpired indicate an entry whether expired.
func (entry *Entry) IsExpired() bool {
	return entry.expire != 0 && entry.expire < time.Now().UnixNano()
}

// CacheStat store cache stats.
type CacheStat struct {
	Entries int64
	Expired int64
	Hits    int64
	Misses  int64
	Total   int64
}

// CacheConfig is configuration struct for local cache.
type CacheConfig struct {
	Expiration time.Duration
	ExpireTick time.Duration
}

// NewCacheConfig populate a default cache config.
func NewCacheConfig() *CacheConfig {
	return &CacheConfig{
		Expiration: defaultExpiration,
		ExpireTick: defaultExpireTick,
	}
}

// LocalCache is an in-memory struct store key-value pairs.
type LocalCache struct {
	data       map[Key]Entry
	mu         sync.RWMutex
	expiration time.Duration
	evicted    func(key Key, value Entry)
	stats      *CacheStat
}

// ResponseEntry is a wrapper of response data.
type ResponseEntry struct {
	Valid bool
	Value interface{}
}

var nilResponse = &ResponseEntry{false, nil}

// NewLocalCache return a empty LocalCache.
func NewLocalCache(config *CacheConfig) *LocalCache {
	if config == nil {
		config = NewCacheConfig()
	}
	lc := &LocalCache{
		data:       make(map[Key]Entry),
		expiration: config.Expiration,
		stats:      &CacheStat{},
	}
	go lc.expireLoop(config.ExpireTick)
	return lc
}

func (c *LocalCache) expireLoop(tick time.Duration) {
	ticker := time.Tick(tick)
	for {
		select {
		case <-ticker:
			c.expireKeys()
		}
	}
}

func (c *LocalCache) expireKeys() {
	c.mu.Lock()
	for key, entry := range c.data {
		if entry.IsExpired() {
			delete(c.data, key)
			if c.evicted != nil {
				c.evicted(key, entry)
			}
		}
	}
	c.mu.Unlock()
}

// SetEvictedFunc set evicted func, this must be called no more once.
func (c *LocalCache) SetEvictedFunc(fn func(Key, Entry)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evicted != nil {
		panic(ErrDuplicateEvictedFunc)
	}
	c.evicted = fn
}

func (c *LocalCache) search(key Key) (entry Entry, ok bool) {
	if entry, ok := c.data[key]; ok {
		if !entry.IsExpired() {
			return entry, true
		}
	}
	return
}

// Add will do same as Set but return an error if key exists.
func (c *LocalCache) Add(key Key, value interface{}) error {
	return c.AddWithExpire(key, value, c.expiration)
}

// AddWithExpire will do same as SetWithExpire but return an error if key exists.
func (c *LocalCache) AddWithExpire(key Key, value interface{}, duration time.Duration) error {
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[Key]Entry)
	}
	_, ok := c.search(key)
	if ok {
		c.mu.Unlock()
		return ErrDuplicateKey
	}
	var e int64
	if duration > 0 {
		e = time.Now().Add(duration).UnixNano()
	}
	c.data[key] = Entry{value: value, expire: e}
	c.stats.Entries++
	c.stats.Total++
	c.mu.Unlock()
	return nil
}

// Set set key-value with default expiration.
func (c *LocalCache) Set(key Key, value interface{}) {
	c.SetWithExpire(key, value, c.expiration)
}

// SetWithExpire set key-value with user setup expiration.
func (c *LocalCache) SetWithExpire(key Key, value interface{}, duration time.Duration) {
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[Key]Entry)
	}
	var e int64
	if duration > 0 {
		e = time.Now().Add(duration).UnixNano()
	}
	c.data[key] = Entry{value: value, expire: e}
	c.stats.Entries++
	c.stats.Total++
	c.mu.Unlock()
}

// Get get the value associated by a key or an error.
func (c *LocalCache) Get(key Key) (v interface{}, err error) {
	v, _, err = c.GetWithExpire(key)
	return
}

// GetWithExpire get the value and left life associated by a key or an error.
func (c *LocalCache) GetWithExpire(key Key) (v interface{}, expire time.Duration, err error) {
	c.mu.RLock()
	if e, ok := c.data[key]; ok {
		now := time.Now()
		if !e.IsExpired() {
			c.stats.Hits++
			c.mu.RUnlock()
			return e.value, time.Duration(e.expire - now.UnixNano()), nil
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
		c.stats.Misses++
		c.mu.RUnlock()
		return nil, ExpireDuration, ErrExpiredKey
	}
	c.stats.Misses++
	c.mu.RUnlock()
	return nil, ExpireDuration, ErrNoSuchKey
}

// GetEntry get a response entry which explain usability of the value or an error.
func (c *LocalCache) GetEntry(key Key) (v *ResponseEntry, err error) {
	c.mu.RLock()
	if e, ok := c.data[key]; ok {
		if !e.IsExpired() {
			c.stats.Hits++
			c.mu.RUnlock()
			return &ResponseEntry{true, e.value}, nil
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
		c.stats.Misses++
		c.mu.RUnlock()
		return nilResponse, ErrExpiredKey
	}
	c.stats.Misses++
	c.mu.RUnlock()
	return nilResponse, ErrNoSuchKey
}

// GetKeysEntry get a map of Key-ResponseEntry which explain usability of the value.
func (c *LocalCache) GetKeysEntry(keys []Key) (v map[Key]*ResponseEntry) {
	v = make(map[Key]*ResponseEntry)
	c.mu.Lock()
	for _, key := range keys {
		if e, ok := c.data[key]; ok {
			if !e.IsExpired() {
				c.stats.Hits++
				v[key] = &ResponseEntry{Valid: true, Value: e.value}
			} else {
				c.stats.Entries--
				c.stats.Expired++
				if c.evicted != nil {
					c.evicted(key, e)
				}
				delete(c.data, key)
				v[key] = nilResponse
				c.stats.Misses++
			}
		} else {
			c.stats.Misses++
			v[key] = nilResponse
		}
	}
	c.mu.Unlock()
	return
}

// GetBool get bool value associated by key or an error.
func (c *LocalCache) GetBool(key Key) (v bool, err error) {
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
func (c *LocalCache) GetInt64(key Key) (v int64, err error) {
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
func (c *LocalCache) GetUint64(key Key) (v uint64, err error) {
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
func (c *LocalCache) GetFloat64(key Key) (v float64, err error) {
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
func (c *LocalCache) GetString(key Key) (v string, err error) {
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
func (c *LocalCache) GetByte(key Key) (v byte, err error) {
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
func (c *LocalCache) GetRune(key Key) (v rune, err error) {
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
func (c *LocalCache) Expire(key Key) (err error) {
	c.mu.Lock()
	if e, ok := c.data[key]; ok {
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		c.stats.Entries--
		c.stats.Expired++
	}
	c.mu.Unlock()
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
	c.data = make(map[Key]Entry)
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
	c.data = make(map[Key]Entry)
	c.stats = &CacheStat{}
	c.mu.Unlock()
}

// Stats return cache stats.
func (c *LocalCache) Stats() *CacheStat {
	c.mu.Lock()
	stats := c.stats
	c.mu.Unlock()
	return stats
}
