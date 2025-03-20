package main

import (
	"fmt"
	"time"
)

func main() {
	backingStore := func(key string) (string, bool) {
		if key == "keyX" {
			return "valueX", true
		}
		return "", false
	}

	var cache Cache[string, string] = NewLRUCache[string, string](2, 5*time.Second, backingStore, &NoOpCacheListener[string]{})

	cache.Put("key1", "value1")
	fmt.Println(cache.Get("key1")) // Expected: value1

	cache.Put("key2", "value2")
	cache.Put("key3", "value3")    // Evicts key1
	fmt.Println(cache.Get("key1")) // Expected: "" (not found)

	fmt.Println(cache.Get("keyX")) // Expected: valueX (from backing store)
}
