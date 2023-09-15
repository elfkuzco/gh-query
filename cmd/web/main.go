package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	ghquery "github.com/elfkuzco/gh-query"
)

type application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
}

type templateData struct {
	Repositories    *[]ghquery.Repository
	LastRepository  *ghquery.Repository // for keeping track of last repository in a fetch operation. used as target for adding infinte scroll htmx listener
	Query           string
	TotalCount      int
	LanguageOptions map[string]string
	SortOptions     map[string]string
	SelectedLang    string
	SelectedSort    string
	NextPage        int
	NextPageUrl     string
}

var resultsPerPage = 50

func humanizeCount(count int) string {
	if count/1000 > 0 {
		return fmt.Sprintf("%.3gK", float64(count)/1000)
	} else if count/1000000 > 0 {
		return fmt.Sprintf("%.3gM", float64(count)/1000000)
	}
	return fmt.Sprintf("%d", count)
}

var functions = template.FuncMap{"humanizeCount": humanizeCount}

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
func (app *application) home(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		app.serverError(w, fmt.Errorf("%s is not allowed for this endpoint", r.Method))
		return
	}
	var td = templateData{
		LanguageOptions: ghquery.LanguageOptions,
		SortOptions:     ghquery.SortOptions,
		SelectedLang:    "",
		SelectedSort:    "stars", // default sort property
		NextPageUrl:     "",
	}

	var results *ghquery.RepositorySearchResult
	q := r.URL.Query().Get("q")

	var ts *template.Template
	var files []string
	var err error

	// Load all the templates or just the template with the
	// results depending on the HTMX request headers
	if r.Header.Get("HX-Request") == "true" {
		files = []string{"./templates/body.partial.tmpl"}
		ts, err = template.New("body.partial.tmpl").Funcs(functions).ParseFiles(files...)
		if err != nil {
			app.serverError(w, err)
			return
		}
	} else {
		files = []string{
			"./templates/home.page.tmpl",
			"./templates/base.layout.tmpl",
			"./templates/body.partial.tmpl",
		}
		ts, err = template.New("home.page.tmpl").Funcs(functions).ParseFiles(files...)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}

	if q != "" {
		// Populate the qualifiers based on the syntax for search
		// on Github. See https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories

		search := []string{q, "in:name"} // Search by repository name
		lang := r.URL.Query().Get("lang")
		if lang != "" {
			search = append(search, fmt.Sprintf("language:%s", lang))
			td.SelectedLang = lang
		}

		query := url.Values{}
		query.Add("q", strings.Join(search, " "))
		query.Add("per_page", fmt.Sprintf("%d", resultsPerPage))

		// Add the other query parameters tot the query string
		sort := r.URL.Query().Get("sort")
		if sort != "" {
			query.Add("sort", sort)
			td.SelectedSort = sort
		}

		// the current page to fetch, default to 1 if page is
		// not a valid number
		var page int
		pg := r.URL.Query().Get("page")
		if pg != "" {
			page, err = strconv.Atoi(pg)
			if err != nil {
				page = 1
			}
		} else {
			page = 1
		}
		query.Add("page", fmt.Sprintf("%d", page))

		results, err = ghquery.FetchRepos(query.Encode())
		if err != nil {
			app.serverError(w, err)
			return
		}

		td.Query = q
		td.Repositories = results.Items
		td.TotalCount = results.TotalCount
		app.infoLog.Printf("found %d repositories for search: '%s'\n", results.TotalCount, query.Encode())
		// Determine the next page to fetch
		numPages := int(math.Ceil(float64(td.TotalCount) / float64(resultsPerPage)))
		if page < numPages {
			td.NextPage = page + 1
			v := *results.Items
			td.LastRepository = &(v[len(v)-1])
			query.Set("page", fmt.Sprintf("%d", td.NextPage))
			td.NextPageUrl = query.Encode()
		}
	}

	var buf = new(bytes.Buffer)
	var tmplErr error

	// Whether to render the full page or a partial template based on htmx headers
	if r.Header.Get("HX-Request") == "true" {
		if r.URL.Query().Get("skip_table_header") != "" {
			tmplErr = ts.ExecuteTemplate(buf, "results", td)
		} else {
			tmplErr = ts.ExecuteTemplate(buf, "body", td)
		}
	} else {
		tmplErr = ts.Execute(buf, td)
	}

	if tmplErr != nil {
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
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	return app.logRequest(mux)
}
