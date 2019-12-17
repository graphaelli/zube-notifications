package main

import (
	"time"
)

type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
	Total      int `json:"total"`
}

type Project struct {
	ID             int          `json:"id"`
	AccountID      int          `json:"account_id"`
	Description    string       `json:"description"`
	Name           string       `json:"name"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
	Slug           string       `json:"slug"`
	Private        bool         `json:"private"`
	PriorityFormat string       `json:"priority_format"`
	Priority       bool         `json:"priority"`
	Points         bool         `json:"points"`
	Triage         bool         `json:"triage"`
	Upvotes        bool         `json:"upvotes"`
	Sources        []Sources    `json:"sources"`
	Workspaces     []Workspace `json:"workspaces"`
}

type Sources struct {
	ID                int       `json:"id"`
	GithubOwnerID     int       `json:"github_owner_id"`
	Description       string    `json:"description"`
	FullName          string    `json:"full_name"`
	Homepage          string    `json:"homepage"`
	HTMLURL           string    `json:"html_url"`
	Name              string    `json:"name"`
	Private           bool      `json:"private"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	WebhookVerifiedAt time.Time `json:"webhook_verified_at"`
	InitialImportAt   time.Time `json:"initial_import_at"`
}

type Workspace struct {
	ID                int       `json:"id"`
	ProjectID         int       `json:"project_id"`
	Description       string    `json:"description"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Private           bool      `json:"private"`
	PriorityFormat    string    `json:"priority_format"`
	Priority          bool      `json:"priority"`
	Points            bool      `json:"points"`
	Upvotes           bool      `json:"upvotes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ArchiveMergedPrs  bool      `json:"archive_merged_prs"`
	UseCategoryLabels bool      `json:"use_category_labels"`
}

type UserSetting struct {
	ID                int       `json:"id"`
	ProjectID         int       `json:"project_id"`
	UserID            int       `json:"user_id"`
	SubscriptionLevel string    `json:"subscription_level"`
	CreatedAt         time.Time `json:"created_at"`
}

type UserPreference map[string]interface{}
