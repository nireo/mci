package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nireo/mci/pb"
)

type HttpServer struct {
	HttpServer      *http.Server
	shutdownTimeout time.Duration
	jm              *JobManager
	pool            *AgentPool
}

// NewServer creates a new Server instance.
func NewServer(
	addr string,
	shutdownTimeout time.Duration,
	jm *JobManager,
	pool *AgentPool,
) *HttpServer {
	hs := &HttpServer{
		shutdownTimeout: shutdownTimeout,
		jm:              jm,
		pool:            pool,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /jobs", hs.listJobs)
	mux.HandleFunc("POST /jobs", hs.createJobHandler)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
		// enforce timeouts to avoid resource leaks
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	hs.HttpServer = httpServer

	return hs
}

// Start runs the server in a new goroutine.
func (s *HttpServer) Start(wg *sync.WaitGroup) <-chan error {
	if wg == nil {
		panic("Server.Start requires a non-nil WaitGroup") // Enforce proper usage
	}

	errChan := make(chan error, 1)
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer close(errChan)

		log.Printf("Server starting and listening on %s\n", s.HttpServer.Addr)
		err := s.HttpServer.ListenAndServe()

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
func (s *HttpServer) Shutdown() error {
	log.Println("Server shutdown requested. Initiating graceful shutdown...")
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	err := s.HttpServer.Shutdown(ctx)
	if err != nil {
		log.Printf("Graceful shutdown failed: %v\n", err)
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	log.Println("Graceful shutdown method completed successfully.")
	return nil
}

func (hs *HttpServer) createJobHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		log.Printf("error reading body: %s", err)
		return
	}
	defer r.Body.Close()

	var data *pb.Job
	err = json.Unmarshal(body, data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		log.Printf("invalid json: %s", err)
		return
	}

	info := hs.jm.AddJob(data)

	if err := hs.pool.SelectAndExecuteJob(context.Background(), info.Job); err != nil {
		http.Error(w, "Error selecting agent", http.StatusInternalServerError)
		log.Printf("invalid select and execute: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (hs *HttpServer) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs := hs.jm.ListJobs()
	json.NewEncoder(w).Encode(jobs)
}
