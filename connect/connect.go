package connect

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// fetchData retrieves data from a given URL and returns a map that can be used with Go templates.
func FetchData(url string) (map[string]interface{}, error) {
	// Perform the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Unmarshal JSON data into a map
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return data, nil
}
