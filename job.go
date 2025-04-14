package mci

import "time"

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusSuccess   JobStatus = "running"
	StatusFailure   JobStatus = "running"
	StatusCancelled JobStatus = "running"
	Statuserror     JobStatus = "running"
)

type Job struct {
	ID         string     `json:"id"`          // Unique identifier for this job run (e.g., UUID)
	Status     JobStatus  `json:"status"`      // Current status of the job
	CreatedAt  time.Time  `json:"created_at"`  // Timestamp when the job was created in our system
	StartedAt  *time.Time `json:"started_at"`  // Timestamp when the job started running (nullable)
	FinishedAt *time.Time `json:"finished_at"` // Timestamp when the job finished (nullable)

	TriggerEvent     string `json:"trigger_event"`      // Value of X-GitHub-Event (e.g., "push", "pull_request")
	GitHubDeliveryID string `json:"github_delivery_id"` // Value of X-GitHub-Delivery (for debugging/tracing)
	PusherName       string `json:"pusher_name"`        // From payload: pusher.name (for push events)
	PusherEmail      string `json:"pusher_email"`       // From payload: pusher.email (for push events)

	RepoGitHubID  int64  `json:"repo_github_id"` // payload: repository.id
	RepoName      string `json:"repo_name"`      // payload: repository.name
	RepoFullName  string `json:"repo_full_name"` // payload: repository.full_name (e.g., "owner/repo")
	RepoOwner     string `json:"repo_owner"`     // payload: repository.owner.login
	RepoPrivate   bool   `json:"repo_private"`   // payload: repository.private
	RepoHTMLURL   string `json:"repo_html_url"`  // payload: repository.html_url (Link to repo)
	RepoCloneURL  string `json:"repo_clone_url"` // payload: repository.clone_url (HTTPS clone URL)
	RepoSSHURL    string `json:"repo_ssh_url"`   // payload: repository.ssh_url (SSH clone URL)
	DefaultBranch string `json:"default_branch"` // payload: repository.default_branch

	Ref             string    `json:"ref"`
	IsBranch        bool      `json:"is_branch"`
	IsTag           bool      `json:"is_tag"`
	BranchOrTagName string    `json:"branch_or_tag_name"`
	CommitSHA       string    `json:"commit_sha"`
	CommitMessage   string    `json:"commit_message"`
	CommitAuthor    string    `json:"commit_author"`
	CommitTimestamp time.Time `json:"commit_timestamp"`
	CompareURL      string    `json:"compare_url"`
}
