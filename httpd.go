package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"rate-limit/internal/middleware"
	"rate-limit/internal/tokenbucket"
)

func runHTTPD(args []string) {
	fs := flag.NewFlagSet("httpd", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	rate := fs.Float64("rate", 10, "sustained requests per second")
	burst := fs.Int("burst", 20, "burst capacity")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	bucket, err := tokenbucket.NewBucket(*rate, *burst, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "httpd: %v\n", err)
		os.Exit(1)
	}
	var mu sync.Mutex

	limiterMW := middleware.New(middleware.Config{
		Limit:  *burst,
		Window: time.Second,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/take", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ok := bucket.TryTake(time.Now(), 1)
		tokens := bucket.Tokens(time.Now())
		mu.Unlock()
		held := HoldTakeLive(ok, tokens)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed": held.Allowed,
			"tokens":  held.Tokens,
		})
	})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tokens := bucket.Tokens(time.Now())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rate":   *rate,
			"burst":  *burst,
			"tokens": tokens,
		})
	})

	handler := limiterMW(mux)
	fmt.Printf("rate-limit httpd on %s (rate=%.1f burst=%d)\n", *addr, *rate, *burst)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		fmt.Fprintf(os.Stderr, "httpd: %v\n", err)
		os.Exit(1)
	}
}
