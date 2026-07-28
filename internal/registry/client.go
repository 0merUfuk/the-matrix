package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// requestTimeout is the maximum duration for any single HTTP request.
	requestTimeout = 5 * time.Second

	// maxBodyBytes is the maximum response body size we will read into memory.
	// Prevents a misbehaving registry from exhausting memory.
	maxBodyBytes = 10 * 1024 * 1024 // 10 MB
)

// ClientVersion can be set by the caller (e.g., cmd/neo/main.go) to include
// the tool version in HTTP requests. Defaults to "dev" for test/library use.
var ClientVersion = "dev"

// userAgent returns the User-Agent header value for registry HTTP requests.
func userAgentString() string {
	return "the-matrix-neo/" + ClientVersion
}

// httpClient returns a configured *http.Client with the standard timeout.
func httpClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}

// doGet performs a GET request with the standard User-Agent header.
// Returns the response body bytes or an error. Callers do not need to close the body.
func doGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgentString())

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", url, err)
	}
	return body, nil
}

// FetchRegistryIndex downloads and parses the registry index from the given URL.
func FetchRegistryIndex(registryURL string) (*RegistryIndex, error) {
	data, err := doGet(registryURL)
	if err != nil {
		return nil, fmt.Errorf("fetching registry index: %w", err)
	}

	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parsing registry index: %w", err)
	}
	return &index, nil
}

// FetchPackDetail downloads and parses a pack's pack.json from the given URL.
func FetchPackDetail(packURL string) (*PackDetail, error) {
	data, err := doGet(packURL)
	if err != nil {
		return nil, fmt.Errorf("fetching pack detail: %w", err)
	}

	var detail PackDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("parsing pack detail: %w", err)
	}
	return &detail, nil
}

// FetchPackFile downloads a single knowledge doc file and returns its content as a string.
func FetchPackFile(fileURL string) (string, error) {
	data, err := doGet(fileURL)
	if err != nil {
		return "", fmt.Errorf("fetching pack file: %w", err)
	}
	return string(data), nil
}
