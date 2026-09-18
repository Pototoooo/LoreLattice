package resultcache

import "testing"

func TestCacheRoundTrip(t *testing.T) {
	cache := New[string]()
	key := Key("tenant-1", "model-1", "payload")
	cache.Set(key, "result")
	if got, ok := cache.Get(key); !ok || got != "result" {
		t.Fatalf("cache Get() = %q, %v", got, ok)
	}
}

func TestKeyPreservesPartBoundaries(t *testing.T) {
	if Key("ab", "c") == Key("a", "bc") {
		t.Fatal("cache key must preserve part boundaries")
	}
}
