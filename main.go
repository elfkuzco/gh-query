package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
}

type RepositorySearchResult struct {
	TotalCount        int  `json:"total_count"`
	IncompleteResults bool `json:"incomplete_results"`
	Items             *[]Repository
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
}

type Owner struct {
	Username  string `json:"login"`
	AvatarUrl string `json:"avatar_url"`
	HTMLUrl   string `json:"url"`
}

type templateData struct {
	Repositories *[]Repository
	Query        string
	TotalCount   int
}

const RepositorySearchUrl = "https://api.github.com/search/repositories"

func main() {
	addr := flag.String("addr", ":5000", "HTTP network address")
	flag.Parse()

	// logger for writing INFO level messages
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	app := &application{
		errorLog: errorLog,
		infoLog:  infoLog,
	}

	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: app.errorLog,
		Handler:  app.routes(),
	}

	app.infoLog.Printf("Running server on port %s\n", *addr)
	err := srv.ListenAndServe()
	errorLog.Fatal(err)
}

func fetchRepos(query string) (*RepositorySearchResult, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", RepositorySearchUrl+"?"+query, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search for query '%s' failed with status: %s", query, resp.Status)
	}

	var result RepositorySearchResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		app.serverError(w, fmt.Errorf("%s is not allowed for this endpoint", r.Method))
		return
	}
	var td templateData
	var buf = new(bytes.Buffer)
	var results *RepositorySearchResult
	q := r.URL.Query().Get("q")

	files := []string{"./templates/home.page.tmpl", "./templates/base.layout.tmpl"}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if q != "" {
		search := []string{q}
		// Populate the qualifiers based on the syntax for search on Github
		lang := r.URL.Query().Get("lang")
		if lang != "" {
			search = append(search, fmt.Sprintf("language:%s", lang))
		}

		query := url.Values{}
		query.Add("q", strings.Join(search, " "))

		// Add the other query parameters tot the query string
		sort := r.URL.Query().Get("sort")
		if sort != "" {
			query.Add("sort", sort)
		}

		results, err = fetchRepos(query.Encode())
		if err != nil {
			app.serverError(w, err)
			return
		}

		td.Query = q
		td.Repositories = results.Items
		td.TotalCount = results.TotalCount
		app.infoLog.Printf("found %d repositories for search: '%s'\n", results.TotalCount, query.Encode())
	}

	err = ts.Execute(buf, td)
	if err != nil {
		app.serverError(w, err)
		return
	}

	buf.WriteTo(w)
}

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.home)
	return app.logRequest(mux)
}
