package localcache

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrTypeMismatch         = errors.New("err: type mismatch")
	ErrNoSuchKey            = errors.New("err: no such key")
	ErrKeyExpired           = errors.New("err: key already expired")
	ErrDuplicateEvictedFunc = errors.New("err: re-set evicted function")
)

const (
	ExpireDuration = time.Duration(-1)
)

type Entry struct {
	value  interface{}
	expire time.Time
}

type Hasher interface {
	Hash() int64
}

type LocalCache struct {
	data       map[string]*Entry
	mu         sync.RWMutex
	expiration time.Duration
	evicted    func(key string, value *Entry)
}

func NewLocalCache(defaultExpiration int64) *LocalCache {
	return &LocalCache{
		data:       make(map[string]*Entry),
		expiration: time.Second * time.Duration(defaultExpiration),
	}
}

func (c *LocalCache) SetEvictedFunc(f func(string, *Entry)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evicted != nil {
		panic(ErrDuplicateEvictedFunc)
	}
	c.evicted = f
}

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
}

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
	c.mu.Unlock()
}

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
		return nil, ErrKeyExpired
	}
	return nil, ErrNoSuchKey
}

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
		return nil, ExpireDuration, ErrKeyExpired
	}
	return nil, ExpireDuration, ErrNoSuchKey
}

func (c *LocalCache) GetBool(key string) (v bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case bool:
				return e.value.(bool), nil
			}
			return false, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return false, ErrKeyExpired
	}
	return false, ErrNoSuchKey
}

func (c *LocalCache) GetInt64(key string) (v int64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case int:
				return int64(e.value.(int)), nil
			case int8:
				return int64(e.value.(int8)), nil
			case int16:
				return int64(e.value.(int16)), nil
			case int32:
				return int64(e.value.(int32)), nil
			case int64:
				return int64(e.value.(int64)), nil
			}
			return 0, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return 0, ErrKeyExpired
	}
	return 0, ErrNoSuchKey
}

func (c *LocalCache) GetUint64(key string) (v uint64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case uint:
				return uint64(e.value.(uint)), nil
			case uint8:
				return uint64(e.value.(uint8)), nil
			case uint16:
				return uint64(e.value.(uint16)), nil
			case uint32:
				return uint64(e.value.(uint32)), nil
			case uint64:
				return uint64(e.value.(uint64)), nil
			}
			return 0, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return 0, ErrKeyExpired
	}
	return 0, ErrNoSuchKey
}

func (c *LocalCache) GetFloat64(key string) (v float64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case float32:
				return float64(e.value.(float64)), nil
			case float64:
				return e.value.(float64), nil
			}
			return 0, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return 0, ErrKeyExpired
	}
	return 0, ErrNoSuchKey
}

func (c *LocalCache) GetString(key string) (v string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case string:
				return e.value.(string), nil
			case []byte:
				return string(e.value.([]byte)), nil
			}
			return "", ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return "", ErrKeyExpired
	}
	return "", ErrNoSuchKey
}

func (c *LocalCache) GetByte(key string) (v byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case byte:
				return e.value.(byte), nil
			case int8:
				return byte(e.value.(int8)), nil
			}
			return 0, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return 0, ErrKeyExpired
	}
	return 0, ErrNoSuchKey
}

func (c *LocalCache) GetRune(key string) (v rune, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if e.expire.IsZero() || e.expire.After(time.Now()) {
			switch e.value.(type) {
			default:
			case rune:
				return e.value.(rune), nil
			}
			return 0, ErrTypeMismatch
		}
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
		return 0, ErrKeyExpired
	}
	return 0, ErrNoSuchKey
}

func (c *LocalCache) Expire(key string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.data[key]; ok {
		if c.evicted != nil {
			c.evicted(key, e)
		}
		delete(c.data, key)
	}
	return
}

func (c *LocalCache) Flush() {
	c.mu.Lock()
	if c.evicted != nil {
		for k, e := range c.data {
			c.evicted(k, e)
		}
	}
	c.data = make(map[string]*Entry)
	c.mu.Unlock()
}

func (c *LocalCache) Size() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int64(len(c.data))
}
