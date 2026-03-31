package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

// Track request completion order
var (
	completionMutex sync.Mutex
	completionOrder []string
)

func main() {
	mux := http.NewServeMux()

	// ===== PIPELINING DEMO ENDPOINTS =====

	mux.HandleFunc("/req1", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req1 STARTED at %s (handler goroutine)\n", r.Proto, startTime.Format("15:04:05.000"))

		time.Sleep(5 * time.Second)

		endTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req1 COMPLETED at %s\n", r.Proto, endTime.Format("15:04:05.000"))

		completionMutex.Lock()
		completionOrder = append(completionOrder, "req1")
		completionMutex.Unlock()

		w.Header().Set("X-Request-ID", "req1")
		w.Write([]byte("Response from /req1 (5 second handler)\n"))
	})

	mux.HandleFunc("/req2", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req2 STARTED at %s (handler goroutine)\n", r.Proto, startTime.Format("15:04:05.000"))

		time.Sleep(2 * time.Second)

		endTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req2 COMPLETED at %s ⚡ (FIRST TO FINISH)\n", r.Proto, endTime.Format("15:04:05.000"))

		completionMutex.Lock()
		completionOrder = append(completionOrder, "req2")
		completionMutex.Unlock()

		w.Header().Set("X-Request-ID", "req2")
		w.Write([]byte("Response from /req2 (2 second handler)\n"))
	})

	mux.HandleFunc("/req3", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req3 STARTED at %s (handler goroutine)\n", r.Proto, startTime.Format("15:04:05.000"))

		time.Sleep(10 * time.Second)

		endTime := time.Now()
		fmt.Printf("[SERVER] [%s] /req3 COMPLETED at %s\n", r.Proto, endTime.Format("15:04:05.000"))

		completionMutex.Lock()
		completionOrder = append(completionOrder, "req3")
		completionMutex.Unlock()

		w.Header().Set("X-Request-ID", "req3")
		w.Write([]byte("Response from /req3 (10 second handler)\n"))
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		completionMutex.Lock()
		defer completionMutex.Unlock()

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Server-side Request Completion Order:\n"))
		for i, req := range completionOrder {
			w.Write([]byte(fmt.Sprintf("%d. %s\n", i+1, req)))
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[SERVER] *** REQUEST CAUGHT *** Path: %s, Proto: %s, RemoteAddr: %s\n",
			r.URL.Path, r.Proto, r.RemoteAddr)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	})

	// HTTP/1.1
	h1Server := &http.Server{
		Addr:         ":8011",
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	// HTTP/2
	h2Server := &http.Server{
		Addr:    ":8022",
		Handler: mux,
		TLSConfig: &tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
		},
	}

	h2Config := &http2.Server{
		MaxConcurrentStreams: 250,
	}

	if err := http2.ConfigureServer(h2Server, h2Config); err != nil {
		log.Fatalf("H2 Config Error: %v", err)
	}

	// HTTP/3  ✅ FIXED ONLY HERE
	h3Server := &http3.Server{
		Addr:    ":8033",
		Handler: mux,
		TLSConfig: &tls.Config{
			NextProtos: []string{"h3"},
		},
	}

	fmt.Println("🚀 Lab is starting...")

	go func() {
		fmt.Println("  -> [H1.1] https://localhost:8011")
		log.Fatal(h1Server.ListenAndServeTLS("server.crt", "server.key"))
	}()

	go func() {
		fmt.Println("  -> [H2]   https://localhost:8022")
		log.Fatal(h2Server.ListenAndServeTLS("server.crt", "server.key"))
	}()

	go func() {
		fmt.Println("  -> [H3]   https://localhost:8033 (UDP)")
		log.Fatal(h3Server.ListenAndServeTLS("server.crt", "server.key"))
	}()

	select {}
}
