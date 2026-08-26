# rate-limit

Persistent rate limiting with token bucket, fixed window, and sliding log
algorithms. Supports in-memory simulation and durable state checkpointing for
crash-consistent recovery. Pure Go, standard library only.

## Build & Test

```bash
export GOTOOLCHAIN=local CGO_ENABLED=0
go build ./...
go test ./...
```

## Usage

### In-memory simulation

```bash
rate-limit sim -algo bucket -rate 2 -burst 5 example/scenario.txt
rate-limit sim -algo fixed  -rate 2 -burst 5 example/scenario.txt
rate-limit sim -algo slide  -rate 2 -burst 5 example/scenario.txt
```

### Persistent mode (checkpoint/recovery)

```bash
# Session 1: process requests and checkpoint state
rate-limit serve -algo bucket -rate 2 -burst 5 -dir ./data -checkpoint example/scenario.txt

# Session 2: state recovered from disk
rate-limit serve -algo bucket -rate 2 -burst 5 -dir ./data example/scenario.txt
```

Scenario file: one request per line, `<RFC3339-timestamp> <count>`. Lines
starting with `#` are comments.

## Directory Structure

```
internal/clock        Clock interface with fake clock for deterministic tests
internal/tokenbucket  Token bucket: continuous refill, burst capacity, state export
internal/window       Fixed window counter, sliding window log, state export
internal/persist      Binary snapshot persistence (magic, version, CRC32 integrity)
internal/store        Persistent store integrating limiters + checkpoint/recovery
```

## Persistence

The `serve` command and `internal/store` package persist limiter state as a
binary snapshot with CRC32 integrity verification. On recovery:

- Valid snapshot: state is restored (token count, window counters, timestamps)
- Corrupt/truncated snapshot: graceful fallback to fresh limiter
- Clock rewind: tokens are not over-granted; the bucket refuses to refill
  backwards

## Example

See `example/scenario.txt` for a sample request timeline.
