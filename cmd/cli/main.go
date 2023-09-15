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
	var lang string // the repository programming language
	var count int   // how many results to return per page
	var page int    // which page of the results to fetch
	var sort string // how to sort the results (default: "stars")
	var name string // name of the repository to fetch

	flag.IntVar(&count, "count", 10, "how many results to return per page. (default: 10)")
	flag.StringVar(&sort, "sort", "stars", "how to sort the results. (default: 'start'")
	flag.StringVar(&lang, "lang", "", "filter results based on programming language")
	flag.StringVar(&name, "name", "", "name of repository to search")
	flag.IntVar(&page, "page", 1, "page of the results to fetch")

	flag.Parse()

	if name == "" {
		log.Fatal("cannot make search for empty repository name")
	}

	search := []string{name, "in:name"} // Default search is by name
	// Ensure that specified language is a valid language
	if lang != "" {
		lang = strings.ToLower(lang)
		// assert that selected language is valid
		_, ok := ghquery.LanguageOptions[lang]
		if !ok {
			log.Fatalf("unknown language '%s'", lang)
		}
		search = append(search, fmt.Sprintf("language:%s", lang))
	}

	query := url.Values{}
	query.Add("q", strings.Join(search, " "))
	query.Add("per_page", fmt.Sprintf("%d", count))

	// Ensure that specified sort is a valid sort order
	if sort != "" {
		sort = strings.ToLower(sort)
		// assert that selected language is valid
		_, ok := ghquery.SortOptions[sort]
		if !ok {
			log.Fatalf("unknown sort order '%s'", sort)
		}
		query.Add("sort", sort)
	}

	query.Add("page", fmt.Sprintf("%d", page))

	results, err := ghquery.FetchRepos(query.Encode())
	if err != nil {
		log.Fatal(err)
	}
	err = tabulateResults(results)
	if err != nil {
		log.Fatal(err)
	}
}

func tabulateResults(results *ghquery.RepositorySearchResult) error {
	var err error
	const format = "%v\t%v\t%v\t%v\t\n"
	tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(tw, format, "Name", "Owner", "Stars", "Issues")
	fmt.Fprintf(tw, format, "------", "------", "------", "------")
	for _, r := range *results.Items {
		fmt.Fprintf(tw,
			format,
			r.Name,
			r.Owner.Username,
			r.Stars,
			r.OpenIssuesCount,
		)
	}
	tw.Flush()
	return err
}
