package core

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/nireo/mci/pb"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var (
	templates *template.Template
	tplerr    error
)

func init() {
	templates, tplerr = template.New("").ParseFS(templateFS, "templates/*.html")
	if tplerr != nil {
		log.Fatalf("FATAL: Failed to parse HTML templates: %v", tplerr)
	}
	log.Println("HTML templates parsed successfully.")
}

type HttpServer struct {
	HttpServer      *http.Server
	shutdownTimeout time.Duration
	jm              *JobManager
	pool            *AgentPool
	logDir          string
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
		logDir:          "logs",
	}
	mux := http.NewServeMux()
	staticFileServer := http.FileServer(http.FS(staticFS))

	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFileServer))
	mux.HandleFunc("GET /jobs", hs.listJobsHandler)
	mux.HandleFunc("GET /jobs/{id}/logs", hs.getJobLogsHandler)
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

func (hs *HttpServer) renderTemplate(w http.ResponseWriter, name string, data any) {
	tmpl := templates.Lookup(name)
	if tmpl == nil {
		log.Printf("error: template: %s not found", name)
		http.Error(w, "Internal Server Error: template not found", http.StatusInternalServerError)
		return
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("error: failed to execute template %s: %v", name, err)
		http.Error(w, "Internal Server Error: Failed to render page", http.StatusInternalServerError)
	}
}

func (hs *HttpServer) createJobHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error: failed to parse form: %v", err)
		hs.renderJobsPageWithError(w, "failed to parse form data")
		return
	}

	repoURL := r.FormValue("repo_url")
	commitSHA := r.FormValue("commit_sha")

	if repoURL == "" || commitSHA == "" {
		log.Println("warn: Missing repo_url or commit_sha in form submission")
		hs.renderJobsPageWithError(w, "Repository URL and Commit SHA are required.")
		return
	}

	job := &pb.Job{
		RepoUrl:   repoURL,
		CommitSha: commitSHA,
	}
	info := hs.jm.AddJob(job)
	log.Printf("INFO: Job created via HTTP: ID=%s, Repo=%s", info.Job.Id, info.Job.RepoUrl)
	go func() {
		err := hs.pool.SelectAndExecuteJob(context.Background(), info.Job)
		if err != nil {
			log.Printf("ERROR: Failed to initially select agent for job %s: %v", info.Job.Id, err)
		} else {
			log.Printf("INFO: Job %s dispatched to an agent.", info.Job.Id)
		}
	}()

	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

func (hs *HttpServer) renderJobsPageWithError(w http.ResponseWriter, formError string) {
	jobs := hs.jm.ListJobs()
	data := map[string]any{
		"Jobs":        jobs,
		"CurrentYear": time.Now().Year(),
		"FormError":   formError,
	}
	hs.renderTemplate(w, "jobs.html", data)
}

func (hs *HttpServer) listJobsHandler(w http.ResponseWriter, r *http.Request) {
	hs.renderJobsPageWithError(w, "")
}

func (hs *HttpServer) getJobLogsHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "Job ID is required", http.StatusBadRequest)
		return
	}
	jobInfo, ok := hs.jm.GetJob(jobID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	var logsContent string
	var logErr error
	logFilePath := filepath.Join(hs.logDir, fmt.Sprintf("%s.log", jobID))
	logBytes, err := os.ReadFile(logFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("WARN: Log file not found for job %s: %s", jobID, logFilePath)
			logsContent = "No log file found."
		} else {
			log.Printf("ERROR: Failed to read log file %s: %v", logFilePath, err)
			logErr = fmt.Errorf("failed to read logs")
		}
	} else {
		logsContent = string(logBytes)
	}

	// No need to include status badge function
	data := map[string]any{
		"JobInfo":     jobInfo,
		"Logs":        logsContent,
		"LogError":    logErr,
		"CurrentYear": time.Now().Year(),
	}
	hs.renderTemplate(w, "job_log.html", data)
}
