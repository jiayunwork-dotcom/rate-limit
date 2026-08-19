package persist

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"rate-limit/internal/tokenbucket"
	"rate-limit/internal/window"
)

func TestSaveAndLoadBucket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	original := &Snapshot{
		Algo: AlgoBucket,
		Bucket: &tokenbucket.BucketState{
			Rate:   5.0,
			Burst:  10.0,
			Tokens: 7.5,
			Last:   now,
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Algo != AlgoBucket {
		t.Fatalf("Algo = %d, want %d", loaded.Algo, AlgoBucket)
	}
	if loaded.Bucket == nil {
		t.Fatal("Bucket is nil")
	}
	if loaded.Bucket.Rate != 5.0 {
		t.Errorf("Rate = %f, want 5.0", loaded.Bucket.Rate)
	}
	if loaded.Bucket.Burst != 10.0 {
		t.Errorf("Burst = %f, want 10.0", loaded.Bucket.Burst)
	}
	if loaded.Bucket.Tokens != 7.5 {
		t.Errorf("Tokens = %f, want 7.5", loaded.Bucket.Tokens)
	}
	if !loaded.Bucket.Last.Equal(now) {
		t.Errorf("Last = %v, want %v", loaded.Bucket.Last, now)
	}
}

func TestSaveAndLoadFixed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	now := time.Date(2025, 3, 1, 8, 30, 0, 0, time.UTC)
	original := &Snapshot{
		Algo: AlgoFixed,
		Fixed: &window.FixedState{
			Limit:  100,
			Window: 10 * time.Second,
			Count:  42,
			Start:  now,
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Algo != AlgoFixed {
		t.Fatalf("Algo = %d, want %d", loaded.Algo, AlgoFixed)
	}
	if loaded.Fixed.Limit != 100 {
		t.Errorf("Limit = %d, want 100", loaded.Fixed.Limit)
	}
	if loaded.Fixed.Window != 10*time.Second {
		t.Errorf("Window = %v, want 10s", loaded.Fixed.Window)
	}
	if loaded.Fixed.Count != 42 {
		t.Errorf("Count = %d, want 42", loaded.Fixed.Count)
	}
	if !loaded.Fixed.Start.Equal(now) {
		t.Errorf("Start = %v, want %v", loaded.Fixed.Start, now)
	}
}

func TestSaveAndLoadSlidingLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	base := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	stamps := []time.Time{
		base.Add(1 * time.Second),
		base.Add(2 * time.Second),
		base.Add(3 * time.Second),
	}
	original := &Snapshot{
		Algo: AlgoSlide,
		Slide: &window.SlidingLogState{
			Limit:  5,
			Window: 10 * time.Second,
			Stamps: stamps,
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Algo != AlgoSlide {
		t.Fatalf("Algo = %d, want %d", loaded.Algo, AlgoSlide)
	}
	if loaded.Slide.Limit != 5 {
		t.Errorf("Limit = %d, want 5", loaded.Slide.Limit)
	}
	if len(loaded.Slide.Stamps) != 3 {
		t.Fatalf("Stamps len = %d, want 3", len(loaded.Slide.Stamps))
	}
	for i, ts := range loaded.Slide.Stamps {
		if !ts.Equal(stamps[i]) {
			t.Errorf("Stamps[%d] = %v, want %v", i, ts, stamps[i])
		}
	}
}

func TestLoadBadMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.snap")
	os.WriteFile(path, []byte("BADMxxxxxxxxxxxxxxxxxxxx"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for bad magic")
	}
}

func TestLoadCorruptCRC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	snap := &Snapshot{
		Algo: AlgoBucket,
		Bucket: &tokenbucket.BucketState{
			Rate: 1.0, Burst: 5.0, Tokens: 3.0, Last: now,
		},
	}
	if err := Save(path, snap); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	// flip a byte in payload area
	data[8] ^= 0xFF
	os.WriteFile(path, data, 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected CRC error")
	}
}

func TestLoadTruncated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	snap := &Snapshot{
		Algo: AlgoBucket,
		Bucket: &tokenbucket.BucketState{
			Rate: 2.0, Burst: 10.0, Tokens: 5.0, Last: now,
		},
	}
	if err := Save(path, snap); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	os.WriteFile(path, data[:len(data)/2], 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected truncation error")
	}
}

func TestLoadUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.snap")

	snap := &Snapshot{
		Algo: AlgoBucket,
		Bucket: &tokenbucket.BucketState{
			Rate: 1.0, Burst: 2.0, Tokens: 1.0,
			Last: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	Save(path, snap)

	data, _ := os.ReadFile(path)
	// version is at offset 4; set to 99
	data[4] = 99
	// recompute CRC
	payload := data[:len(data)-4]
	crc := crc32IEEE(payload)
	binary.LittleEndian.PutUint32(data[len(data)-4:], crc)
	os.WriteFile(path, data, 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected unsupported version error")
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/nonexistent/path/snap.dat")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestSaveAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atomic.snap")

	snap := &Snapshot{
		Algo: AlgoBucket,
		Bucket: &tokenbucket.BucketState{
			Rate: 1.0, Burst: 1.0, Tokens: 1.0,
			Last: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := Save(path, snap); err != nil {
		t.Fatal(err)
	}

	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error(".tmp file should not remain after save")
	}
}

func crc32IEEE(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}
