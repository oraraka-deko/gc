package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkWriteBehind_Set(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "wb_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebbleStore, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "pebble")})
	if err != nil {
		b.Fatal(err)
	}
	defer pebbleStore.Close()

	wb, err := NewWriteBehindStore(pebbleStore, DefaultWriteBehindConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer wb.Close()

	val := []byte(`{"available":1000000000,"status":"active"}`)
	key := []byte("led:acc:user:999:main:XTR")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = wb.Set(key, val, false)
	}
}

func BenchmarkWriteBehind_Get(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "wb_bench_get_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebbleStore, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "pebble")})
	if err != nil {
		b.Fatal(err)
	}
	defer pebbleStore.Close()

	wb, err := NewWriteBehindStore(pebbleStore, DefaultWriteBehindConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer wb.Close()

	key := []byte("led:acc:user:999:main:XTR")
	val := []byte(`{"available":1000000000,"status":"active"}`)
	_ = wb.Set(key, val, false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = wb.Get(key)
	}
}

func BenchmarkWriteBehind_PrefixScan(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "wb_bench_scan_*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pebbleStore, err := NewPebbleStore(PebbleConfig{Path: filepath.Join(tmpDir, "pebble")})
	if err != nil {
		b.Fatal(err)
	}
	defer pebbleStore.Close()

	wb, err := NewWriteBehindStore(pebbleStore, DefaultWriteBehindConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer wb.Close()

	for i := 0; i < 500; i++ {
		k := []byte(fmt.Sprintf("led:acc:user:%d:main:TON", i))
		v := []byte(fmt.Sprintf(`{"available":%d}`, i*1000))
		_ = wb.Set(k, v, false)
	}

	prefix := []byte("led:acc:")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		count := 0
		_ = wb.PrefixScan(prefix, func(k, v []byte) bool {
			count++
			return true
		})
	}
}
