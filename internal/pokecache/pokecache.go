package pokecache

import (
    "sync"
    "time"
)

type cacheEntry struct {
    createdAt time.Time
    val       []byte
}

type Cache struct {
    mu       sync.Mutex
    items    map[string]cacheEntry
    interval time.Duration
}

func (c *Cache) Add(key string, val []byte) {
    c.mu.Lock()
    defer c.mu.Unlock()
    entry := cacheEntry{
        createdAt: time.Now(),
        val:       val,
    }
    c.items[key] = entry
}

func NewCache(interval time.Duration) *Cache {
    c := &Cache{
        items:    make(map[string]cacheEntry),
        interval: interval,
    }
    go c.reapLoop()
    return c
}

func (c *Cache) Get(key string) ([]byte, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    entry, ok := c.items[key]
    if ok {
        return entry.val, true
    }
    return nil, false
}

func (c *Cache) reapLoop() {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()
    for {
        <-ticker.C
        c.mu.Lock()
        for key, entry := range c.items {
            if time.Since(entry.createdAt) > c.interval {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}
