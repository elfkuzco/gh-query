package ghquery

import "time"

type Owner struct {
	Username  string `json:"login"`
	AvatarUrl string `json:"avatar_url"`
	HTMLUrl   string `json:"url"`
}

type Repository struct {
	ID              int
	Name            string
	FullName        string `json:"full_name"`
	Owner           *Owner
	HTMLUrl         string `json:"html_url"`
	Description     string
	Language        string
	OpenIssuesCount int `json:"open_issues_count"`
	Archived        bool
	Disabled        bool
	Private         bool
	CreatedAt       time.Time `json:"created_at"`
	Stars           int       `json:"stargazers_count"`
}

type RepositorySearchResult struct {
	TotalCount        int  `json:"total_count"`
	IncompleteResults bool `json:"incomplete_results"`
	Items             *[]Repository
}

const RepositorySearchUrl = "https://api.github.com/search/repositories"

type Options map[string]string

var LanguageOptions = Options{
	"python":     "Python",
	"javascript": "JavaScript",
	"java":       "Java",
	"c":          "C",
	"cpp":        "C++",
	"ruby":       "Ruby",
	"go":         "Go",
	"swift":      "Swift",
}

var SortOptions = Options{
	"stars":              "Stars",
	"forks":              "Forks",
	"help-wanted-issues": "Help Wanted",
	"updated":            "Updated",
}

var ScopeOptions = Options{
	"name":        "name",
	"description": "description",
	"topics":      "topics",
	"readme":      "readme",
}
