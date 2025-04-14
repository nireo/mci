package mci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// webhook gets information from a given
type githubPushWebhook struct {
	after  string // the SHA of the most recent commit on ref after the push
	before string // the SHA of the most reecnt commit on ref before the push
}

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

// NewServer creates a new Server instance.
func NewServer(addr string, handler http.Handler, shutdownTimeout time.Duration) *Server {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
		// enforce timeouts to avoid resource leaks
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		httpServer:      httpServer,
		shutdownTimeout: shutdownTimeout,
	}
}

// Start runs the server in a new goroutine.
func (s *Server) Start(wg *sync.WaitGroup) <-chan error {
	if wg == nil {
		panic("Server.Start requires a non-nil WaitGroup") // Enforce proper usage
	}

	errChan := make(chan error, 1)
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(errChan)

		log.Printf("Server starting and listening on %s\n", s.httpServer.Addr)
		err := s.httpServer.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v\n", err)
			errChan <- err
		} else if err == http.ErrServerClosed {
			log.Println("Server shutdown gracefully via Shutdown() method.")
		} else {
			log.Println("Server stopped unexpectedly without error.")
		}
		log.Println("Server listening goroutine finished.")
	}()

	return errChan
}

// Shutdown attempts to gracefully shut down the server.
func (s *Server) Shutdown() error {
	log.Println("Server shutdown requested. Initiating graceful shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		log.Printf("Graceful shutdown failed: %v\n", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	log.Println("Graceful shutdown method completed successfully.")
	return nil
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		log.Printf("error reading body: %s", err)
		return
	}
	defer r.Body.Close()

	var data githubPushWebhook
	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		log.Printf("invalid json: %s", err)
		return
	}
}
