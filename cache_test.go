package main

import (
	"testing"
	"time"
)

type CountingCacheListener[K comparable] struct {
	hitMap    map[K]int
	missMap   map[K]int
	evictMap  map[K]int
	expireMap map[K]int
}

func NewCountingCacheListener[K comparable]() *CountingCacheListener[K] {
	return &CountingCacheListener[K]{
		hitMap:    make(map[K]int),
		missMap:   make(map[K]int),
		evictMap:  make(map[K]int),
		expireMap: make(map[K]int),
	}
}

func (l *CountingCacheListener[K]) OnHit(key K) {
	l.hitMap[key]++
}

func (l *CountingCacheListener[K]) OnMiss(key K) {
	l.missMap[key]++
}

func (l *CountingCacheListener[K]) OnEvict(key K) {
	l.evictMap[key]++
}

func (l *CountingCacheListener[K]) OnExpire(key K) {
	l.expireMap[key]++
}

// Helper function to create a new cache with a simple backing store
func newTestCache(capacity int, defaultTTL time.Duration, listener CacheListener[string]) Cache[string, string] {
	backingStore := func(key string) (string, bool) {
		if key == "keyX" {
			return "valueX", true
		}
		return "", false
	}
	return NewLRUCache[string, string](capacity, defaultTTL, backingStore, listener, 5*time.Second)
}

// Test Case 1: Add and Retrieve
func TestCachePutGet(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	cache.Put("key1", "value1")
	if value := cache.Get("key1"); value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}
	if value := listener.hitMap["key1"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
}

// Test Case 2: Retrieve Non-existent Key
func TestCacheGetNonExistentKey(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	if value := cache.Get("keyX"); value != "valueX" { // Comes from backing store
		t.Errorf("Expected 'valueX', got '%s'", value)
	}
	if value := listener.missMap["keyX"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
	if value := cache.Get("keyY"); value != "" { // Not in backing store
		t.Errorf("Expected '', got '%s'", value)
	}
	if value := listener.missMap["keyY"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
}

// Test Case 3: Update Existing Key
func TestCacheUpdateExistingKey(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	cache.Put("key1", "value1")
	cache.Put("key1", "value2")

	if value := cache.Get("key1"); value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}
}

// Test Case 4: Remove Key
func TestCacheRemove(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	cache.Put("key1", "value1")
	cache.Remove("key1")

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
}

// Test Case 5: Eviction Policy
func TestCacheEviction(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3") // This should evict "key1"

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
	if value := listener.evictMap["key1"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
}

// Test Case 6: Expiration of Cached Items
func TestCacheExpiration(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 2*time.Second, listener)

	cache.Put("key1", "value1")
	time.Sleep(3 * time.Second) // Wait for expiration

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
	if value := listener.expireMap["key1"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
}

// Test Case 7: Refresh from Backing Store
func TestCacheRefreshFromBackingStore(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 2*time.Second, listener)

	cache.Put("keyX", "staleValue")
	time.Sleep(3 * time.Second) // Let it expire

	if value := cache.Get("keyX"); value != "valueX" {
		t.Errorf("Expected 'valueX' from backing store, got '%s'", value)
	}
}

// Test Case 8: Key specific expiration of Cached Items
func TestKeySpecificCacheExpiration(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := newTestCache(2, 5*time.Second, listener)

	cache.Put("key1", "value1", 1*time.Second)
	time.Sleep(1 * time.Second) // Wait for expiration

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
	if value := listener.expireMap["key1"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
}

// Test Case 8: Key specific expiration of Cached Items
func TestAutoCleanupByBackgroundThread(t *testing.T) {
	listener := NewCountingCacheListener[string]()
	cache := NewLRUCache[string, string](2, 5*time.Second, nil, listener, 1*time.Second)
	cache.Put("key1", "value1")
	time.Sleep(5 * time.Second) // Wait for expiration
	if value := listener.expireMap["key1"]; value != 1 {
		t.Errorf("Expected '1', got '%d'", value)
	}
	cache.Close()
}
