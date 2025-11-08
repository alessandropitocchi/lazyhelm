// Copyright 2025 Alessandro Pitocchi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package artifacthub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL = "https://artifacthub.io/api/v1"
	// kind 0 = Helm charts
	helmKind = 0
)

// Client is the Artifact Hub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Artifact Hub API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// SearchPackages searches for Helm packages on Artifact Hub
func (c *Client) SearchPackages(query string, limit int) ([]Package, error) {
	if limit == 0 {
		limit = 20
	}

	params := url.Values{}
	params.Add("ts_query_web", query)
	params.Add("facets", "false")
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("kind", fmt.Sprintf("%d", helmKind))

	searchURL := fmt.Sprintf("%s/packages/search?%s", c.baseURL, params.Encode())

	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return searchResp.Packages, nil
}

// GetPackageDetails gets detailed information about a specific package
func (c *Client) GetPackageDetails(repoName, packageName string) (*Package, error) {
	detailURL := fmt.Sprintf("%s/packages/helm/%s/%s", c.baseURL, repoName, packageName)

	resp, err := c.httpClient.Get(detailURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get package details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var pkg Package
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pkg, nil
}

// GetPackageVersion gets a specific version of a package
func (c *Client) GetPackageVersion(repoName, packageName, version string) (*Package, error) {
	versionURL := fmt.Sprintf("%s/packages/helm/%s/%s/%s", c.baseURL, repoName, packageName, version)

	resp, err := c.httpClient.Get(versionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get package version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var pkg Package
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pkg, nil
}
