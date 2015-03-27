# ttlru

[![GoDoc](https://godoc.org/github.com/zvelo/ttlru?status.svg)](https://godoc.org/github.com/zvelo/ttlru) [![Circle CI](https://circleci.com/gh/zvelo/ttlru.svg?style=svg)](https://circleci.com/gh/zvelo/ttlru) [![Coverage Status](https://coveralls.io/repos/zvelo/ttlru/badge.svg)](https://coveralls.io/r/zvelo/ttlru)

Package ttlru provides a simple, goroutine safe, cache with a fixed number of entries. Each entry has a per-cache defined TTL. This TTL is reset on both modification and access of the value. As a result, if the cache is full, and no items have expired, when adding a new item, the item with the soonest expiration will be evicted.

It is based on the LRU implementation in golang-lru:
[github.com/hashicorp/golang-lru](http://godoc.org/github.com/hashicorp/golang-lru)

Which in turn is based on the LRU implementation in groupcache:
[github.com/golang/groupcache/lru](http://godoc.org/github.com/golang/groupcache/lru)
