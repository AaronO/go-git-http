package github

import (
	"time"
)

type GitCommit struct {
	Author    string `json:"author"`
	Email     string `json:"email"`
	Repo      string `json:"repo"`
	RepoOwner string `json:"repo_owner"`
	Message   string `json:"message"`
	Date      string `json:"date"`
	Hash      string `json:"hash"`
}

type GitHubCommit struct {
	Distinct  bool         `json:"distinct"`
	URL       string       `json:"url"`
	Id        string       `json:"id"`
	Timestamp time.Time    `json:"timestamp"`
	Added     []string     `json:"added"`
	Message   string       `json:"message"`
	Committer GitHubPerson `json:"committer"`
	Author    GitHubPerson `json:"author"`
	Modified  []string     `json:"modified"`
	Removed   []string     `json:"removed"`
}

type GitHubRepo struct {
	HasIssues    bool         `json:"has_issues"`
	HasWiki      bool         `json:"has_wiki"`
	Size         int          `json:"size"`
	Description  string       `json:"description"`
	Owner        GitHubPerson `json:"owner"`
	Homepage     string       `json:"homepage"`
	Watchers     int          `json:"watchers"`
	Language     string       `json:"language"`
	PushedAt     time.Time    `json:"pushed_at"`
	Name         string       `json:"name"`
	Organization string       `json:"organization"`
	HasDownloads bool         `json:"has_downloads"`
	CreatedAt    time.Time    `json:"created_at"`
	URL          string       `json:"url"`
	OpenIssues   int          `json:"open_issues"`
	Forks        int          `json:"forks"`
	Private      bool         `json:"private"`
	Fork         bool         `json:"fork"`
	Stargazers   int          `json:"stargazers"`
}

type GitHubPerson struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Username *string `json:"username"`
}

type GitHubPayload struct {
	Before     string         `json:"before"`
	Created    bool           `json:"created"`
	Ref        string         `json:"ref"`
	Deleted    bool           `json:"deleted"`
	After      string         `json:"after"`
	HeadCommit GitHubCommit   `json:"head_commit"`
	Commits    []GitHubCommit `json:"commits"`
	Repository GitHubRepo     `json:"repository"`
	Forced     bool           `json:"forced"`
	Compare    string         `json:"compare"`
	Pusher     GitHubPerson   `json:"pusher"`
}

type PayloadPong struct {
	Ok    bool   `json:"ok"`
	Event string `json:"event"`
	Error error  `json:"error,omitempty"`
}
