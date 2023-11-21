package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start with basic request info
		log.Printf("Time: %s, Method: %s, URL: %s, RemoteAddr: %s\n", time.Now().Format(time.RFC3339), r.Method, r.URL, r.RemoteAddr)

		// Log headers in a readable format
		log.Println("Headers:")
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("  %s: %s\n", name, value)
			}
		}

		// Log the body if present
		if r.Body != nil && r.Header.Get("Content-Type") != "multipart/form-data" {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body.Close() // must close
			if len(bodyBytes) > 0 {
				log.Printf("Body: %s\n", string(bodyBytes))
			} else {
				log.Println("Body: [Empty]")
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Log form values if present
		if err := r.ParseForm(); err == nil {
			if len(r.Form) > 0 {
				log.Println("Form Values:")
				for key, values := range r.Form {
					for _, value := range values {
						log.Printf("  %s: %s\n", key, value)
					}
				}
			} else {
				log.Println("Form Values: [None]")
			}
		}

		log.Println()
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Generate a timestamp for the filename
	timestamp := time.Now().Format("2006-01-02_15-04-05") // YYYY-MM-DD_hh-mm-ss
	logFilename := "requests_" + timestamp + ".log"

	// Set up log file
	logFile, err := os.OpenFile(logFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %s\n", err)
	}
	defer logFile.Close()

	// Set log output to file
	log.SetOutput(logFile)

	// Set up static file server with logger
	fileServer := http.FileServer(http.Dir("./static"))
	loggedFS := Logger(fileServer)

	http.Handle("/", loggedFS)
	server := &http.Server{Addr: ":8080"}

	go func() {
		log.Println("Starting USER server on port 8080")

		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Error starting USER server: %s\n", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interrupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-c
	log.Printf("Got signal: %s, exiting.", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	server.Shutdown(ctx)
}
