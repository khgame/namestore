package namestore

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestMemory_Get_DoubleCheck_KeyDeletedBetweenReads tests the race where
// a key is deleted between the optimistic read and the write lock acquisition.
func TestMemory_Get_DoubleCheck_KeyDeletedBetweenReads(t *testing.T) {
	// Run multiple iterations with high concurrency to increase chance of hitting edge case
	for iteration := 0; iteration < 1000; iteration++ {
		d := NewInMemoryDriver().(*Memory)
		ctx := context.Background()

		// Create a key that will expire very soon.
		_ = d.Set(ctx, "race-key", []byte("value"), 1*time.Millisecond)

		var wg sync.WaitGroup
		numReaders := 50 // High concurrency
		numDeleters := 50

		// Multiple readers trying to hit the slow path
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				// Stagger the reads slightly
				time.Sleep(time.Duration(id%3) * time.Microsecond)
				_, _ = d.Get(ctx, "race-key")
			}(i)
		}

		// Multiple deleters trying to delete the key
		for i := 0; i < numDeleters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%3) * time.Microsecond)
				d.mu.Lock()
				delete(d.data, "race-key")
				d.mu.Unlock()
			}(i)
		}

		wg.Wait()
	}
}

// TestMemory_Get_DoubleCheck_KeyResetBetweenReads tests the race where
// a key is reset (set again) between the optimistic read and write lock.
func TestMemory_Get_DoubleCheck_KeyResetBetweenReads(t *testing.T) {
	// Run multiple iterations with high concurrency
	for iteration := 0; iteration < 1000; iteration++ {
		d := NewInMemoryDriver().(*Memory)
		ctx := context.Background()

		// Create a key that expires very soon.
		_ = d.Set(ctx, "reset-key", []byte("old-value"), 1*time.Nanosecond)

		time.Sleep(10 * time.Microsecond) // Let it expire.

		var wg sync.WaitGroup
		numReaders := 50
		numSetters := 50

		// Multiple readers trying to read expired key (will enter slow path).
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%5) * time.Microsecond)
				_, _ = d.Get(ctx, "reset-key")
			}(i)
		}

		// Multiple setters resetting the key with new value.
		for i := 0; i < numSetters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%5) * time.Microsecond)
				_ = d.Set(ctx, "reset-key", []byte("new-value"), 0)
			}(i)
		}

		wg.Wait()
	}
}

// TestMemory_Exists_DoubleCheck_KeyDeletedBetweenReads tests similar race for Exists.
func TestMemory_Exists_DoubleCheck_KeyDeletedBetweenReads(t *testing.T) {
	// Run multiple iterations with high concurrency
	for iteration := 0; iteration < 1000; iteration++ {
		d := NewInMemoryDriver().(*Memory)
		ctx := context.Background()

		// Create a key that will expire very soon.
		_ = d.Set(ctx, "exists-race", []byte("value"), 1*time.Millisecond)

		var wg sync.WaitGroup
		numCheckers := 50
		numDeleters := 50

		// Multiple checkers trying to check existence
		for i := 0; i < numCheckers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%3) * time.Microsecond)
				_, _ = d.Exists(ctx, "exists-race")
			}(i)
		}

		// Multiple deleters
		for i := 0; i < numDeleters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%3) * time.Microsecond)
				d.mu.Lock()
				delete(d.data, "exists-race")
				d.mu.Unlock()
			}(i)
		}

		wg.Wait()
	}
}

// TestMemory_Exists_DoubleCheck_KeyResetBetweenReads tests key reset race for Exists.
func TestMemory_Exists_DoubleCheck_KeyResetBetweenReads(t *testing.T) {
	// Run multiple iterations with high concurrency
	for iteration := 0; iteration < 1000; iteration++ {
		d := NewInMemoryDriver().(*Memory)
		ctx := context.Background()

		// Create a key that expires very soon.
		_ = d.Set(ctx, "exists-reset", []byte("value"), 1*time.Nanosecond)

		time.Sleep(10 * time.Microsecond) // Let it expire.

		var wg sync.WaitGroup
		numCheckers := 50
		numSetters := 50

		// Multiple checkers
		for i := 0; i < numCheckers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%5) * time.Microsecond)
				_, _ = d.Exists(ctx, "exists-reset")
			}(i)
		}

		// Multiple setters
		for i := 0; i < numSetters; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				time.Sleep(time.Duration(id%5) * time.Microsecond)
				_ = d.Set(ctx, "exists-reset", []byte("new-value"), 0)
			}(i)
		}

		wg.Wait()
	}
}

// TestMemory_Get_RaceCondition tests the double-check optimization in Get.
func TestMemory_Get_RaceCondition(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Setup: key that will expire soon.
	_ = d.Set(ctx, "key1", []byte("value1"), 5*time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Keep reading.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = d.Get(ctx, "key1")
			time.Sleep(100 * time.Microsecond)
		}
	}()

	// Goroutine 2: Wait for expiration then try to read.
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Wait for key to expire.
		_, err := d.Get(ctx, "key1")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound for expired key, got %v", err)
		}
	}()

	wg.Wait()
}

// TestMemory_Exists_RaceCondition tests the double-check optimization in Exists.
func TestMemory_Exists_RaceCondition(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Setup: key that will expire soon.
	_ = d.Set(ctx, "key1", []byte("value1"), 5*time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Keep checking existence.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = d.Exists(ctx, "key1")
			time.Sleep(100 * time.Microsecond)
		}
	}()

	// Goroutine 2: Wait for expiration then check.
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Wait for key to expire.
		exists, _ := d.Exists(ctx, "key1")
		if exists {
			t.Error("Expected key to not exist after expiration")
		}
	}()

	wg.Wait()
}

// TestMemory_Get_DoubleCheckPath tests the double-check slow path.
func TestMemory_Get_DoubleCheckPath(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Create a key that expires very soon.
	_ = d.Set(ctx, "expiring", []byte("value"), 1*time.Nanosecond)

	// Wait to ensure it's expired.
	time.Sleep(2 * time.Millisecond)

	// This should trigger the slow path (double-check).
	_, err := d.Get(ctx, "expiring")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// Verify key was deleted.
	d.mu.Lock()
	_, exists := d.data["expiring"]
	d.mu.Unlock()

	if exists {
		t.Error("Expired key should have been deleted")
	}
}

// TestMemory_Exists_DoubleCheckPath tests the double-check slow path.
func TestMemory_Exists_DoubleCheckPath(t *testing.T) {
	d := NewInMemoryDriver().(*Memory)
	ctx := context.Background()

	// Create a key that expires very soon.
	_ = d.Set(ctx, "expiring", []byte("value"), 1*time.Nanosecond)

	// Wait to ensure it's expired.
	time.Sleep(2 * time.Millisecond)

	// This should trigger the slow path (double-check).
	exists, _ := d.Exists(ctx, "expiring")
	if exists {
		t.Error("Expected false for expired key")
	}

	// Verify key was deleted.
	d.mu.Lock()
	_, keyExists := d.data["expiring"]
	d.mu.Unlock()

	if keyExists {
		t.Error("Expired key should have been deleted")
	}
}

