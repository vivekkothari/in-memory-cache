package main

import (
	"fmt"
	cache2 "github.com/vivekkothari/in-memory-cache/cache"
	"time"
)

func main() {
	backingStore := func(key string) (string, bool) {
		if key == "keyX" {
			return "valueX", true
		}
		return "", false
	}

	var cache cache2.Cache[string, string] = cache2.NewLRUCache[string, string](2, 5*time.Second, backingStore, nil, 5*time.Second)

	cache.Put("key1", "value1")
	fmt.Println(cache.Get("key1")) // Expected: value1

	cache.Put("key2", "value2")
	cache.Put("key3", "value3")    // Evicts key1
	fmt.Println(cache.Get("key1")) // Expected: "" (not found)

	fmt.Println(cache.Get("keyX")) // Expected: valueX (from backing store)

	cache.Close()
}
