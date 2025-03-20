package main

import (
	"testing"
	"time"
)

// Helper function to create a new cache with a simple backing store
func newTestCache(capacity int, defaultTTL time.Duration) Cache[string, string] {
	backingStore := func(key string) (string, bool) {
		if key == "keyX" {
			return "valueX", true
		}
		return "", false
	}
	return NewLRUCache[string, string](capacity, defaultTTL, backingStore, &NoOpCacheListener[string]{})
}

// Test Case 1: Add and Retrieve
func TestCachePutGet(t *testing.T) {
	cache := newTestCache(2, 5*time.Second)

	cache.Put("key1", "value1")
	if value := cache.Get("key1"); value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}
}

// Test Case 2: Retrieve Non-existent Key
func TestCacheGetNonExistentKey(t *testing.T) {
	cache := newTestCache(2, 5*time.Second)

	if value := cache.Get("keyX"); value != "valueX" { // Comes from backing store
		t.Errorf("Expected 'valueX', got '%s'", value)
	}
	if value := cache.Get("keyY"); value != "" { // Not in backing store
		t.Errorf("Expected '', got '%s'", value)
	}
}

// Test Case 3: Update Existing Key
func TestCacheUpdateExistingKey(t *testing.T) {
	cache := newTestCache(2, 5*time.Second)

	cache.Put("key1", "value1")
	cache.Put("key1", "value2")

	if value := cache.Get("key1"); value != "value2" {
		t.Errorf("Expected 'value2', got '%s'", value)
	}
}

// Test Case 4: Remove Key
func TestCacheRemove(t *testing.T) {
	cache := newTestCache(2, 5*time.Second)

	cache.Put("key1", "value1")
	cache.Remove("key1")

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
}

// Test Case 5: Eviction Policy
func TestCacheEviction(t *testing.T) {
	cache := newTestCache(2, 5*time.Second)

	cache.Put("key1", "value1")
	cache.Put("key2", "value2")
	cache.Put("key3", "value3") // This should evict "key1"

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
}

// Test Case 6: Expiration of Cached Items
func TestCacheExpiration(t *testing.T) {
	cache := newTestCache(2, 2*time.Second)

	cache.Put("key1", "value1")
	time.Sleep(3 * time.Second) // Wait for expiration

	if value := cache.Get("key1"); value != "" {
		t.Errorf("Expected '', got '%s'", value)
	}
}

// Test Case 7: Refresh from Backing Store
func TestCacheRefreshFromBackingStore(t *testing.T) {
	cache := newTestCache(2, 2*time.Second)

	cache.Put("keyX", "staleValue")
	time.Sleep(3 * time.Second) // Let it expire

	if value := cache.Get("keyX"); value != "valueX" {
		t.Errorf("Expected 'valueX' from backing store, got '%s'", value)
	}
}
