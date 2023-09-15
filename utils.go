package ghquery

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Fetches the repositories that the query string matches.
// See https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories
// on how to construct the query string

func FetchRepos(query string) (*RepositorySearchResult, error) {
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
