// Package persist implements binary snapshot persistence for rate limiter state.
//
// File format (little-endian):
//
//	[magic 4B]["RLIM"] [version 1B][0x01] [algo 1B]
//	[payload ...]
//	[crc32 4B] (IEEE over everything before this field)
//
// Algo codes:
//
//	0x01 = TokenBucket
//	0x02 = FixedWindow
//	0x03 = SlidingLog
//
// TokenBucket payload:
//
//	[rate float64] [burst float64] [tokens float64] [last_unix_ns int64]
//
// FixedWindow payload:
//
//	[limit uint32] [window_ns int64] [count uint32] [start_unix_ns int64]
//
// SlidingLog payload:
//
//	[limit uint32] [window_ns int64] [stamp_count uint32]
//	for each stamp: [unix_ns int64]
package persist

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"os"
	"time"

	"rate-limit/internal/tokenbucket"
	"rate-limit/internal/window"
)

var fileMagic = [4]byte{'R', 'L', 'I', 'M'}

const currentVersion = 1

const (
	AlgoBucket byte = 0x01
	AlgoFixed  byte = 0x02
	AlgoSlide  byte = 0x03
)

var (
	ErrBadMagic           = errors.New("persist: invalid magic bytes")
	ErrUnsupportedVersion = errors.New("persist: unsupported version")
	ErrCorrupt            = errors.New("persist: data corrupted (CRC mismatch)")
	ErrTruncated          = errors.New("persist: file truncated")
	ErrUnknownAlgo        = errors.New("persist: unknown algorithm code")
)

// Snapshot holds the decoded state from a persistence file.
type Snapshot struct {
	Algo   byte
	Bucket *tokenbucket.BucketState
	Fixed  *window.FixedState
	Slide  *window.SlidingLogState
}

// Save writes a snapshot to path atomically (write tmp + rename).
func Save(path string, snap *Snapshot) error {
	data, err := encode(snap)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("persist: create tmp: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmp)
	}()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("persist: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("persist: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("persist: close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("persist: rename: %w", err)
	}
	return nil
}

// Load reads and decodes a snapshot from path.
func Load(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persist: read: %w", err)
	}
	return decode(data)
}

func encode(snap *Snapshot) ([]byte, error) {
	buf := make([]byte, 0, 256)

	// header
	buf = append(buf, fileMagic[:]...)
	buf = append(buf, currentVersion)
	buf = append(buf, snap.Algo)

	switch snap.Algo {
	case AlgoBucket:
		if snap.Bucket == nil {
			return nil, fmt.Errorf("persist: bucket state is nil")
		}
		buf = appendFloat64(buf, snap.Bucket.Rate)
		buf = appendFloat64(buf, snap.Bucket.Burst)
		buf = appendFloat64(buf, snap.Bucket.Tokens)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(snap.Bucket.Last.UnixNano()))

	case AlgoFixed:
		if snap.Fixed == nil {
			return nil, fmt.Errorf("persist: fixed state is nil")
		}
		buf = binary.LittleEndian.AppendUint32(buf, uint32(snap.Fixed.Limit))
		buf = binary.LittleEndian.AppendUint64(buf, uint64(snap.Fixed.Window))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(snap.Fixed.Count))
		buf = binary.LittleEndian.AppendUint64(buf, uint64(snap.Fixed.Start.UnixNano()))

	case AlgoSlide:
		if snap.Slide == nil {
			return nil, fmt.Errorf("persist: slide state is nil")
		}
		buf = binary.LittleEndian.AppendUint32(buf, uint32(snap.Slide.Limit))
		buf = binary.LittleEndian.AppendUint64(buf, uint64(snap.Slide.Window))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(snap.Slide.Stamps)))
		for _, ts := range snap.Slide.Stamps {
			buf = binary.LittleEndian.AppendUint64(buf, uint64(ts.UnixNano()))
		}

	default:
		return nil, fmt.Errorf("persist: unknown algo %d", snap.Algo)
	}

	// CRC over all data so far
	crc := crc32.ChecksumIEEE(buf)
	buf = binary.LittleEndian.AppendUint32(buf, crc)
	return buf, nil
}

func decode(data []byte) (*Snapshot, error) {
	// minimum: magic(4) + version(1) + algo(1) + crc(4) = 10
	if len(data) < 10 {
		return nil, ErrTruncated
	}

	// verify CRC first
	payload := data[:len(data)-4]
	storedCRC := binary.LittleEndian.Uint32(data[len(data)-4:])
	if crc32.ChecksumIEEE(payload) != storedCRC {
		return nil, ErrCorrupt
	}

	pos := 0

	// magic
	var m [4]byte
	copy(m[:], data[pos:pos+4])
	pos += 4
	if m != fileMagic {
		return nil, ErrBadMagic
	}

	// version
	ver := data[pos]
	pos++
	if ver != currentVersion {
		return nil, fmt.Errorf("%w: got %d", ErrUnsupportedVersion, ver)
	}

	// algo
	algo := data[pos]
	pos++

	snap := &Snapshot{Algo: algo}
	rest := payload[pos:]

	switch algo {
	case AlgoBucket:
		if len(rest) < 32 {
			return nil, ErrTruncated
		}
		rate := readFloat64(rest[0:8])
		burst := readFloat64(rest[8:16])
		tokens := readFloat64(rest[16:24])
		lastNs := int64(binary.LittleEndian.Uint64(rest[24:32]))
		snap.Bucket = &tokenbucket.BucketState{
			Rate:   rate,
			Burst:  burst,
			Tokens: tokens,
			Last:   time.Unix(0, lastNs),
		}

	case AlgoFixed:
		if len(rest) < 20 {
			return nil, ErrTruncated
		}
		limit := binary.LittleEndian.Uint32(rest[0:4])
		winNs := int64(binary.LittleEndian.Uint64(rest[4:12]))
		count := binary.LittleEndian.Uint32(rest[12:16])
		startNs := int64(binary.LittleEndian.Uint64(rest[16:24]))
		snap.Fixed = &window.FixedState{
			Limit:  int(limit),
			Window: time.Duration(winNs),
			Count:  int(count),
			Start:  time.Unix(0, startNs),
		}

	case AlgoSlide:
		if len(rest) < 12 {
			return nil, ErrTruncated
		}
		limit := binary.LittleEndian.Uint32(rest[0:4])
		winNs := int64(binary.LittleEndian.Uint64(rest[4:12]))
		stampCount := binary.LittleEndian.Uint32(rest[12:16])
		if len(rest) < 16+int(stampCount)*8 {
			return nil, ErrTruncated
		}
		stamps := make([]time.Time, stampCount)
		off := 16
		for i := range stamps {
			ns := int64(binary.LittleEndian.Uint64(rest[off : off+8]))
			stamps[i] = time.Unix(0, ns)
			off += 8
		}
		snap.Slide = &window.SlidingLogState{
			Limit:  int(limit),
			Window: time.Duration(winNs),
			Stamps: stamps,
		}

	default:
		return nil, ErrUnknownAlgo
	}

	return snap, nil
}

func appendFloat64(buf []byte, f float64) []byte {
	bits := math.Float64bits(f)
	return binary.LittleEndian.AppendUint64(buf, bits)
}

func readFloat64(b []byte) float64 {
	bits := binary.LittleEndian.Uint64(b)
	return math.Float64frombits(bits)
}
