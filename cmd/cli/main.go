package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	ghquery "github.com/elfkuzco/gh-query"
)

func main() {
	// Definition of search query parameters
	var lang string  // the repository programming language
	var count int    // how many results to return per page
	var page int     // which page of the results to fetch
	var sort string  // how to sort the results (default: "stars")
	var name string  // name of the repository to fetch
	var scope string // restrict search to name, description, topics or readme
	var order string // how to sort the query results
	var showRepoURL bool

	flag.IntVar(&count, "count", 10, "how many results to return per page.")
	flag.StringVar(&sort, "sort", "stars",
		"how to sort the results. Supported values are: stars, forks, help-wanted-issues, updated.",
	)
	flag.StringVar(&lang, "lang", "", "filter results based on programming. language")
	flag.StringVar(&name, "name", "", "name of repository to search.")
	flag.IntVar(&page, "page", 1, "page of the results to fetch.")
	flag.StringVar(&scope, "scope", "",
		"restrict search to the repository name, description, topics, or contents of README. Supported values are: name, description, topics, readme",
	)
	flag.StringVar(&order, "order", "desc",
		"how to sort the search results. Supported values are: asc, desc",
	)
	flag.BoolVar(&showRepoURL, "show-repo-url", false, "show repository url in results")

	flag.Parse()

	if name == "" {
		log.Fatal("cannot make search for empty repository name")
	}

	// Build up optional search parameters
	search := []string{name}
	// Ensure scope is a valid scope if specified
	if scope != "" {
		scope = strings.ToLower(scope)
		if _, ok := ghquery.ScopeOptions[scope]; !ok {
			log.Fatalf("unknown scope '%s'", scope)
		}
		search = append(search, fmt.Sprintf("in:%s", scope))
	}

	// Ensure that specified language is a valid language
	if lang != "" {
		lang = strings.ToLower(lang)
		if _, ok := ghquery.LanguageOptions[lang]; !ok {
			log.Fatalf("unknown language '%s'", lang)
		}
		search = append(search, fmt.Sprintf("language:%s", lang))
	}

	// Build up query string and parameters
	query := url.Values{}
	query.Add("q", strings.Join(search, " "))

	order = strings.ToLower(order)
	if order == "asc" || order == "desc" {
		query.Add("order", order)
	} else {
		log.Fatalf("unknown order: '%s'", order)
	}

	query.Add("per_page", fmt.Sprintf("%d", count))

	// Ensure that specified sort is a valid sort order
	if sort != "" {
		sort = strings.ToLower(sort)
		if _, ok := ghquery.SortOptions[sort]; !ok {
			log.Fatalf("unknown sort order '%s'", sort)
		}
		query.Add("sort", sort)
	}

	query.Add("page", fmt.Sprintf("%d", page))

	results, err := ghquery.FetchRepos(query.Encode())
	if err != nil {
		log.Fatal(err)
	}
	err = tabulateResults(results, showRepoURL)
	if err != nil {
		log.Fatal(err)
	}
}

func tabulateResults(results *ghquery.RepositorySearchResult, showRepoURL bool) error {
	if results.TotalCount == 0 {
		fmt.Fprint(os.Stdout, "Query did not return any results")
		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	var format = "%v\t%v\t%v\t%v\t\n"

	fmt.Fprintf(tw, format, "Name", "Owner", "Stars", "Issues")
	fmt.Fprintf(tw, format, "------", "------", "------", "------")

	if showRepoURL {
		format := "%v [%v]\t%v\t%v\t%v\t\n"
		for _, r := range *results.Items {
			fmt.Fprintf(tw,
				format,
				r.Name,
				r.HTMLUrl,
				r.Owner.Username,
				r.Stars,
				r.OpenIssuesCount,
			)
		}
	} else {
		for _, r := range *results.Items {
			fmt.Fprintf(tw,
				format,
				r.Name,
				r.Owner.Username,
				r.Stars,
				r.OpenIssuesCount,
			)
		}
	}
	return tw.Flush()
}
