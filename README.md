# a high-performance and gc-friendly LRU cache

[![godoc][godoc-img]][godoc] [![release][release-img]][release] [![goreport][goreport-img]][goreport] [![codecov][codecov-img]][codecov]

### Features

* Simple
    - No Dependencies.
    - 100% code coverage.
    - Less than 1000 lines of Go code.
* Fast
    - Outperforms well-known *LRU* caches.
    - Zero memory allocations .
* GC friendly
    - Pointerless data structs.
    - Continuous memory layout.
* Memory efficient
    - Adds only 26 extra bytes per cache object.
    - Minimized memory usage compared to others.
* Feature optional
    - SlidingCache via `WithSilding(true)` option.
    - LoadingCache via `WithLoader(func(key K) (v V, ttl time.Duration, err error))` option.

### Getting Started

An out of box example. https://go.dev/play/p/01hUdKwp2MC
```go
package main

import (
	"time"

	"github.com/phuslu/lru"
)

func main() {
	cache := lru.New[string, int](8192)

	cache.Set("a", 1, 2*time.Second)
	println(cache.Get("a"))

	time.Sleep(1 * time.Second)
	println(cache.Get("a"))

	time.Sleep(2 * time.Second)
	println(cache.Get("a"))
}
```

Using as a sliding cache. https://go.dev/play/p/l6GUlrAigJK
```go
package main

import (
	"time"

	"github.com/phuslu/lru"
)

func main() {
	cache := lru.New[string, int](8192, lru.WithSilding(true))

	cache.Set("foobar", 42, 3*time.Second)

	time.Sleep(2 * time.Second)
	println(cache.Get("foobar"))

	time.Sleep(2 * time.Second)
	println(cache.Get("foobar"))

	time.Sleep(2 * time.Second)
	println(cache.Get("foobar"))
}
```

Create a loading cache. https://go.dev/play/p/Ve8o4Ihrdxp
```go
package main

import (
	"time"

	"github.com/phuslu/lru"
)

func main() {
	loader := func(key string) (int, time.Duration, error) {
		return 42, time.Hour, nil
	}

	cache := lru.New[string, int](8192, lru.WithLoader(loader))

	println(cache.Get("b"))
	println(cache.GetOrLoad("b"))
	println(cache.Get("b"))
}
```

### Benchmarks

A Performance result as below. Check github [actions][actions] for more results and details.
<details>
  <summary>benchmark on keysize=16, itemsize=1000000, cachesize=50%, concurrency=32 with randomly read (90%) / write(10%)</summary>

  ```go
  // go test -v -cpu=8 -run=none -bench=. -benchtime=5s -benchmem bench_test.go
  package bench

  import (
  	"fmt"
  	"testing"
  	"time"
  	_ "unsafe"

  	theine "github.com/Yiling-J/theine-go"
  	"github.com/cespare/xxhash/v2"
  	cloudflare "github.com/cloudflare/golibs/lrucache"
  	ristretto "github.com/dgraph-io/ristretto"
  	freelru "github.com/elastic/go-freelru"
  	lxzan "github.com/lxzan/memorycache"
  	otter "github.com/maypok86/otter"
  	ecache "github.com/orca-zhang/ecache"
  	phuslu "github.com/phuslu/lru"
  )

  const (
  	keysize     = 16
  	cachesize   = 1000000
  	parallelism = 32
  	writeradio  = 0.1
  )

  var keys = func() (x []string) {
  	x = make([]string, cachesize)
  	for i := 0; i < cachesize; i++ {
  		x[i] = fmt.Sprintf(fmt.Sprintf("%%0%dd", keysize), i)
  	}
  	return
  }()

  //go:noescape
  //go:linkname fastrandn runtime.fastrandn
  func fastrandn(x uint32) uint32

  func BenchmarkCloudflareGetSet(b *testing.B) {
  	cache := cloudflare.NewMultiLRUCache(1024, cachesize/1024)
  	for i := 0; i < cachesize/2; i++ {
  		cache.Set(keys[i], i, time.Now().Add(time.Hour))
  	}
  	b.SetParallelism(parallelism)
  	b.ResetTimer()
  	b.RunParallel(func(pb *testing.PB) {
  		expires := time.Now().Add(time.Hour)
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.Set(keys[i], i, expires)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }

  func BenchmarkEcacheGetSet(b *testing.B) {
  	cache := ecache.NewLRUCache(1024, cachesize/1024, time.Hour)
  	for i := 0; i < cachesize/2; i++ {
  		cache.Put(keys[i], i)
  	}
  	b.SetParallelism(parallelism)
  	b.ResetTimer()
  	b.RunParallel(func(pb *testing.PB) {
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.Put(keys[i], i)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }

  func BenchmarkLxzanGetSet(b *testing.B) {
    cache := lxzan.New[string, int](
    lxzan.WithBucketNum(128),
    lxzan.WithBucketSize(cachesize/128, cachesize/128),
    lxzan.WithInterval(time.Hour, time.Hour),
  )
    for i := 0; i < cachesize/2; i++ {
      cache.Set(keys[i], i, time.Hour)
    }

    b.SetParallelism(parallelism)
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
      waterlevel := int(float32(cachesize) * writeradio)
      for pb.Next() {
        i := int(fastrandn(cachesize))
        if i <= waterlevel {
          cache.Set(keys[i], i, time.Hour)
        } else {
          cache.Get(keys[i])
        }
      }
    })
  }

  func hashStringXXHASH(s string) uint32 {
    return uint32(xxhash.Sum64String(s))
  }

  func BenchmarkFreelruGetSet(b *testing.B) {
    cache, _ := freelru.NewSharded[string, int](cachesize, hashStringXXHASH)
    for i := 0; i < cachesize/2; i++ {
      cache.AddWithLifetime(keys[i], i, time.Hour)
    }

    b.SetParallelism(parallelism)
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
      waterlevel := int(float32(cachesize) * writeradio)
      for pb.Next() {
        i := int(fastrandn(cachesize))
        if i <= waterlevel {
          cache.AddWithLifetime(keys[i], i, time.Hour)
        } else {
          cache.Get(keys[i])
        }
      }
    })
  }

  func BenchmarkRistrettoGetSet(b *testing.B) {
  	cache, _ := ristretto.NewCache(&ristretto.Config{
  		NumCounters: cachesize, // number of keys to track frequency of (10M).
  		MaxCost:     2 << 30,   // maximum cost of cache (2GB).
  		BufferItems: 64,        // number of keys per Get buffer.
  	})
  	for i := 0; i < cachesize/2; i++ {
  		cache.SetWithTTL(keys[i], i, 1, time.Hour)
  	}

  	b.SetParallelism(parallelism)
  	b.ResetTimer()

  	b.RunParallel(func(pb *testing.PB) {
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.SetWithTTL(keys[i], i, 1, time.Hour)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }

  func BenchmarkTheineGetSet(b *testing.B) {
  	cache, _ := theine.NewBuilder[string, int](cachesize).Build()
  	for i := 0; i < cachesize/2; i++ {
  		cache.SetWithTTL(keys[i], i, 1, time.Hour)
  	}

  	b.SetParallelism(parallelism)
  	b.ResetTimer()

  	b.RunParallel(func(pb *testing.PB) {
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.SetWithTTL(keys[i], i, 1, time.Hour)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }

  func BenchmarkOtterGetSet(b *testing.B) {
  	cache, _ := otter.MustBuilder[string, int](cachesize).WithVariableTTL().Build()
  	for i := 0; i < cachesize/2; i++ {
  		cache.Set(keys[i], i, time.Hour)
  	}

  	b.SetParallelism(parallelism)
  	b.ResetTimer()

  	b.RunParallel(func(pb *testing.PB) {
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.Set(keys[i], i, time.Hour)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }

  func BenchmarkPhusluGetSet(b *testing.B) {
  	cache := phuslu.New[string, int](cachesize)
  	for i := 0; i < cachesize/2; i++ {
  		cache.Set(keys[i], i, time.Hour)
  	}

  	b.SetParallelism(parallelism)
  	b.ResetTimer()

  	b.RunParallel(func(pb *testing.PB) {
  		waterlevel := int(float32(cachesize) * writeradio)
  		for pb.Next() {
  			i := int(fastrandn(cachesize))
  			if i <= waterlevel {
  				cache.Set(keys[i], i, time.Hour)
  			} else {
  				cache.Get(keys[i])
  			}
  		}
  	})
  }
  ```
</details>

```
goos: linux
goarch: amd64
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkCloudflareGetSet
BenchmarkCloudflareGetSet-8     37905620         156.7 ns/op        16 B/op        1 allocs/op
BenchmarkEcacheGetSet
BenchmarkEcacheGetSet-8         51791524         116.4 ns/op         2 B/op        0 allocs/op
BenchmarkLxzanGetSet
BenchmarkLxzanGetSet-8          49494284         127.3 ns/op         0 B/op        0 allocs/op
BenchmarkFreelruGetSet
BenchmarkFreelruGetSet-8        56382706         108.7 ns/op         0 B/op        0 allocs/op
BenchmarkRistrettoGetSet
BenchmarkRistrettoGetSet-8      34761458         146.6 ns/op        27 B/op        1 allocs/op
BenchmarkTheineGetSet
BenchmarkTheineGetSet-8         37194024         166.1 ns/op         0 B/op        0 allocs/op
BenchmarkOtterGetSet
BenchmarkOtterGetSet-8          31980354         191.4 ns/op         6 B/op        0 allocs/op
BenchmarkPhusluGetSet
BenchmarkPhusluGetSet-8         67774108          84.88 ns/op        0 B/op        0 allocs/op
PASS
ok    command-line-arguments  60.874s
```

### Memory usage

The Memory usage result as below. Check github [actions][actions] for more results and details.
<details>
  <summary>memory usage on keysize=16(string), valuesize=8(int), itemsize=1000000(1M), cachesize=100%</summary>

  ```go
  // memusage.go
  package main

  import (
  	"fmt"
  	"os"
  	"runtime"
  	"time"

  	theine "github.com/Yiling-J/theine-go"
  	"github.com/cespare/xxhash/v2"
  	cloudflare "github.com/cloudflare/golibs/lrucache"
  	freelru "github.com/elastic/go-freelru"
  	ristretto "github.com/dgraph-io/ristretto"
  	lxzan "github.com/lxzan/memorycache"
  	otter "github.com/maypok86/otter"
  	ecache "github.com/orca-zhang/ecache"
  	phuslu "github.com/phuslu/lru"
  )

  const (
  	keysize   = 16
  	cachesize = 1000000
  )

  var keys []string 

  func main() {
  	keys = make([]string, cachesize)
  	for i := 0; i < cachesize; i++ {
  		keys[i] = fmt.Sprintf(fmt.Sprintf("%%0%dd", keysize), i)
  	}

  	var o runtime.MemStats
  	runtime.ReadMemStats(&o)

  	name := os.Args[1]
  	switch name {
  	case "phuslu":
  		SetupPhuslu()
  	case "freelru":
  		SetupFreelru()
  	case "ristretto":
  		SetupRistretto()
  	case "otter":
  		SetupOtter()
  	case "lxzan":
  		SetupLxzan()
  	case "ecache":
  		SetupEcache()
  	case "cloudflare":
  		SetupCloudflare()
  	case "theine":
  		SetupTheine()
  	default:
  		panic("no cache name")
  	}

  	var m runtime.MemStats
  	runtime.ReadMemStats(&m)

  	fmt.Printf("%s\t%v MiB\t%v MiB\t%v MiB\n",
  		name,
  		(m.Alloc-o.Alloc)/1048576,
  		(m.TotalAlloc-o.TotalAlloc)/1048576,
  		(m.Sys-o.Sys)/1048576,
  	)
  }

  func SetupPhuslu() {
  	cache := phuslu.New[string, int](cachesize)
  	for i := 0; i < cachesize; i++ {
  		cache.Set(keys[i], i, time.Hour)
  	}
  }

  func SetupFreelru() {
  	cache, _ := freelru.NewSharded[string, int](cachesize, func(s string) uint32 { return uint32(xxhash.Sum64String(s)) })
  	for i := 0; i < cachesize; i++ {
  		cache.AddWithLifetime(keys[i], i, time.Hour)
  	}
  }

  func SetupOtter() {
  	cache, _ := otter.MustBuilder[string, int](cachesize).WithVariableTTL().Build()
  	for i := 0; i < cachesize; i++ {
  		cache.Set(keys[i], i, time.Hour)
  	}
  }

  func SetupEcache() {
  	cache := ecache.NewLRUCache(1024, cachesize/1024, time.Hour)
  	for i := 0; i < cachesize; i++ {
  		cache.Put(keys[i], i)
  	}
  }

  func SetupRistretto() {
  	cache, _ := ristretto.NewCache(&ristretto.Config{
  		NumCounters: cachesize,
  		MaxCost:     2 << 30,
  		BufferItems: 64,
  	})
  	for i := 0; i < cachesize; i++ {
  		cache.SetWithTTL(keys[i], i, 1, time.Hour)
  	}
  }

  func SetupLxzan() {
  	cache := lxzan.New[string, int](
		lxzan.WithBucketNum(128),
		lxzan.WithBucketSize(cachesize/128, cachesize/128),
		lxzan.WithInterval(time.Hour, time.Hour),
	)
  	for i := 0; i < cachesize; i++ {
  		cache.Set(keys[i], i, time.Hour)
  	}
  }

  func SetupTheine() {
  	cache, _ := theine.NewBuilder[string, int](cachesize).Build()
  	for i := 0; i < cachesize; i++ {
  		cache.SetWithTTL(keys[i], i, 1, time.Hour)
  	}
  }

  func SetupCloudflare() {
  	cache := cloudflare.NewMultiLRUCache(1024, cachesize/1024)
  	for i := 0; i < cachesize; i++ {
  		cache.Set(keys[i], i, time.Now().Add(time.Hour))
  	}
  }
  ```
</details>

| MemStats   | Alloc   | TotalAlloc | Sys     |
| ---------- | ------- | ---------- | ------- |
| phuslu     | 48 MiB  | 56 MiB     | 57 MiB  |
| lxzan      | 95 MiB  | 103 MiB    | 106 MiB |
| freelru    | 112 MiB | 120 MiB    | 122 MiB |
| ecache     | 123 MiB | 131 MiB    | 127 MiB |
| ristretto  | 171 MiB | 253 MiB    | 181 MiB |
| otter      | 137 MiB | 211 MiB    | 181 MiB |
| theine     | 177 MiB | 223 MiB    | 194 MiB |
| cloudflare | 183 MiB | 191 MiB    | 189 MiB |

### License
LRU is licensed under the MIT License. See the LICENSE file for details.

### Contact
For inquiries or support, contact phus.lu@gmail.com or raise github issues.

[godoc-img]: http://img.shields.io/badge/godoc-reference-blue.svg
[godoc]: https://pkg.go.dev/github.com/phuslu/lru
[release-img]: https://img.shields.io/github/v/tag/phuslu/lru?label=release
[release]: https://github.com/phuslu/lru/tags
[goreport-img]: https://goreportcard.com/badge/github.com/phuslu/lru
[goreport]: https://goreportcard.com/report/github.com/phuslu/lru
[actions]: https://github.com/phuslu/lru/actions/workflows/benchmark.yml
[codecov-img]: https://codecov.io/gh/phuslu/lru/graph/badge.svg?token=Q21AMQNM1K
[codecov]: https://codecov.io/gh/phuslu/lru
