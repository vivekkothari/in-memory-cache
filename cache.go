package main

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type CacheListener[K comparable] interface {
	OnHit(key K)
	OnMiss(key K)
	OnEvict(key K)
	OnExpire(key K)
}

type NoOpCacheListener[K comparable] struct {
}

func (c *NoOpCacheListener[K]) OnHit(key K) {
	fmt.Println(key, "Hit")
}

func (c *NoOpCacheListener[K]) OnMiss(key K) {
	fmt.Println(key, "Miss")
}

func (c *NoOpCacheListener[K]) OnEvict(key K) {
	fmt.Println(key, "Evicted")
}

func (c *NoOpCacheListener[K]) OnExpire(key K) {
	fmt.Println(key, "Expired")
}

type Cache[K comparable, V any] interface {
	Put(key K, value V, ttl ...time.Duration)
	Get(key K) V
	Remove(key K)
}

type CacheItem[K comparable, V any] struct {
	key       K
	value     V
	timestamp time.Time
	expiry    time.Duration
}

type LRUCache[K comparable, V any] struct {
	capacity      int
	cache         map[K]*list.Element
	order         *list.List
	mutex         sync.RWMutex
	defaultTTL    time.Duration
	backingStore  func(K) (V, bool)
	cacheListener CacheListener[K]
}

func NewLRUCache[K comparable, V any](capacity int, defaultTTL time.Duration, backingStore func(K) (V, bool), cacheListener CacheListener[K]) *LRUCache[K, V] {
	var listener CacheListener[K]
	if cacheListener == nil {
		listener = &NoOpCacheListener[K]{}
	} else {
		listener = cacheListener
	}
	var refillStore func(K) (V, bool)
	if backingStore == nil {
		refillStore = func(key K) (V, bool) {
			var zeroV V
			return zeroV, false
		}
	} else {
		refillStore = backingStore
	}
	return &LRUCache[K, V]{
		capacity:      capacity,
		cache:         make(map[K]*list.Element),
		order:         list.New(),
		defaultTTL:    defaultTTL,
		backingStore:  refillStore,
		cacheListener: listener,
	}
}

func (c *LRUCache[K, V]) Put(key K, value V, ttl ...time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiry := c.defaultTTL
	if len(ttl) > 0 {
		expiry = ttl[0]
	}

	if elem, found := c.cache[key]; found {
		c.order.MoveToFront(elem)
		item := elem.Value.(*CacheItem[K, V])
		item.value = value
		item.timestamp = time.Now()
		item.expiry = expiry
		return
	}

	if len(c.cache) >= c.capacity {
		c.evict()
	}

	item := &CacheItem[K, V]{key, value, time.Now(), expiry}
	elem := c.order.PushFront(item)
	c.cache[key] = elem
}

func (c *LRUCache[K, V]) Get(key K) V {
	c.mutex.RLock()

	if elem, found := c.cache[key]; found {
		c.cacheListener.OnHit(key)
		item := elem.Value.(*CacheItem[K, V])
		if time.Since(item.timestamp) > item.expiry {
			c.cacheListener.OnExpire(item.key)
			c.order.Remove(elem)
			delete(c.cache, key)
			c.mutex.RUnlock()
			return c.fetchFromBackingStore(key)
		}
		c.order.MoveToFront(elem)
		item.timestamp = time.Now()
		value := item.value
		c.mutex.RUnlock()
		return value
	}

	c.cacheListener.OnMiss(key)
	c.mutex.RUnlock()
	return c.fetchFromBackingStore(key)
}

func (c *LRUCache[K, V]) Remove(key K) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if elem, found := c.cache[key]; found {
		c.order.Remove(elem)
		delete(c.cache, key)
	}
}

func (c *LRUCache[K, V]) evict() {
	if elem := c.order.Back(); elem != nil {
		item := elem.Value.(*CacheItem[K, V])
		delete(c.cache, item.key)
		c.cacheListener.OnEvict(item.key)
		c.order.Remove(elem)
	}
}

func (c *LRUCache[K, V]) fetchFromBackingStore(key K) V {
	var zeroValue V
	if value, found := c.backingStore(key); found {
		c.Put(key, value)
		return value
	}
	return zeroValue
}
