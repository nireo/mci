package core

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nireo/mci/pb"
)

type JobInfo struct {
	Job       *pb.Job      `json:"job"`
	Status    pb.JobStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type JobManager struct {
	jobs  map[string]*JobInfo
	mutex sync.RWMutex
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*JobInfo),
	}
}

func (jm *JobManager) AddJob(job *pb.Job) *JobInfo {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	if job.Id == "" {
		job.Id = uuid.NewString()
	}

	now := time.Now()
	jobInfo := &JobInfo{
		Job:       job,
		Status:    pb.JobStatus_PENDING,
		CreatedAt: now,
		UpdatedAt: now,
	}
	jm.jobs[job.Id] = jobInfo
	log.Printf("Job added: ID=%s, Repo=%s", job.Id, job.RepoUrl)

	return jobInfo
}

func (jm *JobManager) UpdateJobStatus(jobID string, status pb.JobStatus) error {
	jm.mutex.Lock()
	defer jm.mutex.Unlock()

	jobInfo, ok := jm.jobs[jobID]
	if !ok {
		return fmt.Errorf("job with ID %s not found", jobID)
	}
	jobInfo.Status = status
	jobInfo.UpdatedAt = time.Now()
	log.Printf("Job status updated: ID=%s, Status=%s", jobID, status)
	return nil
}

func (jm *JobManager) GetJob(jobID string) (*JobInfo, bool) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()
	jobInfo, ok := jm.jobs[jobID]

	return jobInfo, ok
}

func (jm *JobManager) ListJobs() []*JobInfo {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	list := make([]*JobInfo, 0, len(jm.jobs))
	for _, jobInfo := range jm.jobs {
		list = append(list, jobInfo)
	}
	return list
}
