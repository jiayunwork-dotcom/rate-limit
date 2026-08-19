package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"rate-limit/internal/store"
	"rate-limit/internal/tokenbucket"
	"rate-limit/internal/window"
)

type limiter interface {
	Try(now time.Time, n int) bool
	State(now time.Time) string
}

type bucketLimiter struct{ b *tokenbucket.Bucket }

func (l bucketLimiter) Try(now time.Time, n int) bool { return l.b.TryTake(now, n) }
func (l bucketLimiter) State(now time.Time) string {
	return fmt.Sprintf("tokens=%.2f", l.b.Tokens(now))
}

type fixedLimiter struct{ w *window.Fixed }

func (l fixedLimiter) Try(now time.Time, n int) bool {
	ok := true
	for i := 0; i < n; i++ {
		if !l.w.Allow(now) {
			ok = false
		}
	}
	return ok && n > 0
}
func (l fixedLimiter) State(now time.Time) string {
	return fmt.Sprintf("count=%d", l.w.Count(now))
}

type slideLimiter struct{ w *window.SlidingLog }

func (l slideLimiter) Try(now time.Time, n int) bool {
	ok := true
	for i := 0; i < n; i++ {
		if !l.w.Allow(now) {
			ok = false
		}
	}
	return ok && n > 0
}
func (l slideLimiter) State(now time.Time) string {
	return fmt.Sprintf("len=%d", l.w.Len(now))
}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type request struct {
	ts    time.Time
	count int
}

func parseScenario(path string) ([]request, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var reqs []request
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		fields := strings.Fields(s)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s:%d: want \"<RFC3339> <count>\", got %q", path, line, s)
		}
		ts, err := time.Parse(time.RFC3339, fields[0])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: bad timestamp %q", path, line, fields[0])
		}
		n, err := strconv.Atoi(fields[1])
		if err != nil || n < 1 {
			return nil, fmt.Errorf("%s:%d: bad count %q", path, line, fields[1])
		}
		reqs = append(reqs, request{ts: ts, count: n})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return reqs, nil
}

func newLimiter(algo string, rate float64, burst int, now time.Time) (limiter, error) {
	switch algo {
	case "bucket":
		b, err := tokenbucket.NewBucket(rate, burst, now)
		if err != nil {
			return nil, err
		}
		return bucketLimiter{b: b}, nil
	case "fixed":
		windowLen := time.Duration(float64(time.Second) * float64(burst) / rate)
		w, err := window.NewFixed(burst, windowLen, now)
		if err != nil {
			return nil, err
		}
		return fixedLimiter{w: w}, nil
	case "slide":
		windowLen := time.Duration(float64(time.Second) * float64(burst) / rate)
		w, err := window.NewSlidingLog(burst, windowLen)
		if err != nil {
			return nil, err
		}
		return slideLimiter{w: w}, nil
	default:
		return nil, fmt.Errorf("unknown algorithm %q (want bucket, fixed or slide)", algo)
	}
}

func runSim(args []string) {
	fs := flag.NewFlagSet("sim", flag.ExitOnError)
	algo := fs.String("algo", "bucket", "bucket | fixed | slide")
	rate := fs.Float64("rate", 2, "sustained requests per second (must be > 0)")
	burst := fs.Int("burst", 5, "burst capacity (must be >= 1)")
	fail(fs.Parse(args))
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: rate-limit sim [-algo A] [-rate R] [-burst B] <scenario-file>")
		os.Exit(2)
	}
	reqs, err := parseScenario(fs.Arg(0))
	fail(err)
	if len(reqs) == 0 {
		fail(fmt.Errorf("scenario has no requests"))
	}
	l, err := newLimiter(*algo, *rate, *burst, reqs[0].ts)
	fail(err)
	allowed, denied := 0, 0
	for _, r := range reqs {
		ok := l.Try(r.ts, r.count)
		status := "ALLOW"
		if !ok {
			status = "DENY "
			denied++
		} else {
			allowed++
		}
		fmt.Printf("%s take=%d %s %s\n", r.ts.Format(time.RFC3339), r.count, status, l.State(r.ts))
	}
	fmt.Printf("summary: %d allowed, %d denied\n", allowed, denied)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	algo := fs.String("algo", "bucket", "bucket | fixed | slide")
	rate := fs.Float64("rate", 2, "sustained requests per second")
	burst := fs.Int("burst", 5, "burst capacity")
	dir := fs.String("dir", "", "data directory for state persistence")
	checkpoint := fs.Bool("checkpoint", false, "write checkpoint after operations")
	winDur := fs.Duration("window", 0, "window duration (fixed/slide); computed from rate+burst if 0")
	fail(fs.Parse(args))
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "serve requires -dir")
		os.Exit(2)
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: rate-limit serve [-algo A] [-rate R] [-burst B] [-window D] -dir DIR <scenario-file>")
		os.Exit(2)
	}

	reqs, err := parseScenario(fs.Arg(0))
	fail(err)
	if len(reqs) == 0 {
		fail(fmt.Errorf("scenario has no requests"))
	}

	cfg := store.Config{
		Algo:   *algo,
		Rate:   *rate,
		Burst:  *burst,
		Window: *winDur,
	}
	s, err := store.Open(*dir, cfg, reqs[0].ts)
	fail(err)
	defer s.Close()

	if s.WasRecovered {
		fmt.Fprintln(os.Stderr, "info: state recovered from checkpoint")
	}

	for _, r := range reqs {
		ok := s.TryTake(r.ts, r.count)
		status := "ALLOW"
		if !ok {
			status = "DENY "
		}
		fmt.Printf("%s take=%d %s\n", r.ts.Format(time.RFC3339), r.count, status)
	}

	if *checkpoint {
		lastTs := reqs[len(reqs)-1].ts
		if err := s.Checkpoint(lastTs); err != nil {
			fail(err)
		}
		fmt.Println("checkpoint: ok")
	}

	allowed, denied := s.Stats()
	fmt.Printf("summary: %d allowed, %d denied\n", allowed, denied)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: rate-limit <command> [flags] <args>

commands:
  sim    replay a scenario through an in-memory limiter
  serve  replay with persistent state (checkpoint/recovery)

flags:
  -algo     bucket | fixed | slide (default bucket)
  -rate     sustained requests per second (default 2)
  -burst    burst capacity (default 5)
  -dir      data directory (serve only)
  -window   window duration for fixed/slide (default: computed from rate+burst)
  -checkpoint  write state to disk after processing`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "sim":
		runSim(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
