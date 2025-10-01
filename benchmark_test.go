package namestore

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

// Benchmark basic operations.

func BenchmarkMemory_Set(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}
}

func BenchmarkMemory_Get(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Get(ctx, fmt.Sprintf("key:%d", i%1000))
	}
}

func BenchmarkMemory_Delete(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Pre-populate keys.
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Delete(ctx, fmt.Sprintf("key:%d", i))
	}
}

func BenchmarkMemory_SetNX(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.SetNX(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}
}

func BenchmarkMemory_Exists(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Exists(ctx, fmt.Sprintf("key:%d", i%1000))
	}
}

// Benchmark batch operations.

func BenchmarkMemory_MGet(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.MGet(ctx, keys)
	}
}

func BenchmarkMemory_MSet(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	pairs := make(map[string][]byte, 10)
	for i := 0; i < 10; i++ {
		pairs[fmt.Sprintf("key:%d", i)] = value
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.MSet(ctx, pairs, 0)
	}
}

func BenchmarkMemory_MDel(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Pre-populate all keys.
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			_ = d.Set(ctx, fmt.Sprintf("key:%d:%d", i, j), value, 0)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keys := make([]string, 10)
		for j := 0; j < 10; j++ {
			keys[j] = fmt.Sprintf("key:%d:%d", i, j)
		}
		_ = d.MDel(ctx, keys)
	}
}

// Benchmark TTL operations.

func BenchmarkMemory_TTL(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.TTL(ctx, fmt.Sprintf("key:%d", i%1000))
	}
}

func BenchmarkMemory_Expire(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Expire(ctx, fmt.Sprintf("key:%d", i%1000), 1*time.Hour)
	}
}

func BenchmarkMemory_Persist(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys with TTL.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Persist(ctx, fmt.Sprintf("key:%d", i%1000))
	}
}

// Benchmark atomic operations.

func BenchmarkMemory_Incr(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Incr(ctx, "counter", 1)
	}
}

func BenchmarkMemory_Decr(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()

	// Setup: initialize counter.
	_, _ = d.Incr(ctx, "counter", 1000000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Decr(ctx, "counter", 1)
	}
}

func BenchmarkMemory_GetSet(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: initialize key.
	_ = d.Set(ctx, "key", value, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.GetSet(ctx, "key", value)
	}
}

func BenchmarkMemory_CompareAndSwap(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	oldValue := []byte("old-value")
	newValue := []byte("new-value")

	// Pre-populate keys.
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), oldValue, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.CompareAndSwap(ctx, fmt.Sprintf("key:%d", i), oldValue, newValue, 0)
	}
}

// Benchmark namespace operations.

func BenchmarkMemory_Keys(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("prefix:key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Keys(ctx, "prefix", "*")
	}
}

func BenchmarkMemory_Clear(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Pre-populate all iterations.
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100; j++ {
			_ = d.Set(ctx, fmt.Sprintf("prefix:%d:key:%d", i, j), value, 0)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Clear(ctx, fmt.Sprintf("prefix:%d", i))
	}
}

// Benchmark client operations (with namespace overhead).

func BenchmarkClient_Set(b *testing.B) {
	c := New[string]("bench", "test")
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Set(ctx, "key", value, 0)
	}
}

func BenchmarkClient_Get(b *testing.B) {
	c := New[string]("bench", "test")
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup.
	_ = c.Set(ctx, "key", value, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get(ctx, "key")
	}
}

func BenchmarkClient_MGet(b *testing.B) {
	c := New[string]("bench", "test")
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 10; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("key:%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.MGet(ctx, keys...)
	}
}

// Benchmark concurrency scenarios.

func BenchmarkMemory_ConcurrentReads(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = d.Get(ctx, fmt.Sprintf("key:%d", i%1000))
			i++
		}
	})
}

func BenchmarkMemory_ConcurrentWrites(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
			i++
		}
	})
}

func BenchmarkMemory_ConcurrentMixed(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("benchmark-value")

	// Setup: populate with keys.
	for i := 0; i < 1000; i++ {
		_ = d.Set(ctx, fmt.Sprintf("key:%d", i), value, 0)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				_, _ = d.Get(ctx, fmt.Sprintf("key:%d", i%1000))
			} else {
				_ = d.Set(ctx, fmt.Sprintf("key:%d", i%1000), value, 0)
			}
			i++
		}
	})
}

// Benchmark different value sizes.

func BenchmarkMemory_Set_SmallValue(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("small")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, "key", value, 0)
	}
}

func BenchmarkMemory_Set_MediumValue(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := make([]byte, 1024) // 1KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, "key", value, 0)
	}
}

func BenchmarkMemory_Set_LargeValue(b *testing.B) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := make([]byte, 1024*1024) // 1MB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.Set(ctx, "key", value, 0)
	}
}

// Benchmark key construction overhead.

func BenchmarkClient_KeyConstruction(b *testing.B) {
	c := New[string]("benchmark", "domain").(*client[string])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.key(fmt.Sprintf("key:%d", i))
	}
}

func BenchmarkClient_KeyConstruction_Simple(b *testing.B) {
	c := New[string]("benchmark", "domain").(*client[string])

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.key("simple-key")
	}
}

// Benchmark integer conversion for atomic operations.

func BenchmarkAtomicOps_IntConversion(b *testing.B) {
	value := int64(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := make([]byte, 8)
		_ = buf
		_ = value
	}
}

// Benchmark with different key count scenarios.

func BenchmarkMemory_Keys_10(b *testing.B) {
	benchmarkKeysWithCount(b, 10)
}

func BenchmarkMemory_Keys_100(b *testing.B) {
	benchmarkKeysWithCount(b, 100)
}

func BenchmarkMemory_Keys_1000(b *testing.B) {
	benchmarkKeysWithCount(b, 1000)
}

func BenchmarkMemory_Keys_10000(b *testing.B) {
	benchmarkKeysWithCount(b, 10000)
}

func benchmarkKeysWithCount(b *testing.B, count int) {
	d := NewInMemoryDriver()
	ctx := context.Background()
	value := []byte("v")

	// Setup: populate with keys.
	for i := 0; i < count; i++ {
		_ = d.Set(ctx, fmt.Sprintf("prefix:key:%d", i), value, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.Keys(ctx, "prefix", "*")
	}
}

// Benchmark string conversion for keys.

func BenchmarkStringConversion_Sprintf(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("key:%d", i)
	}
}

func BenchmarkStringConversion_Itoa(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = "key:" + strconv.Itoa(i)
	}
}
