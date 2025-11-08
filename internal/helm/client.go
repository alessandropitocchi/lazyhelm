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

package helm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

type Client struct {
	settings *cli.EnvSettings
}

func NewClient() *Client {
	return &Client{
		settings: cli.New(),
	}
}

type Repository struct {
	Name string
	URL  string
}

func (c *Client) ListRepositories() ([]Repository, error) {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Repository{}, nil
		}
		return nil, err
	}

	repos := make([]Repository, 0, len(f.Repositories))
	for _, r := range f.Repositories {
		repos = append(repos, Repository{
			Name: r.Name,
			URL:  r.URL,
		})
	}

	return repos, nil
}

type Chart struct {
	Name        string
	Version     string
	Description string
}

func (c *Client) SearchCharts(repoName string) ([]Chart, error) {
	args := []string{"search", "repo", repoName, "--output", "json"}
	
	cmd := exec.Command("helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("helm search failed: %w", err)
	}
	
	var results []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, err
	}
	
	charts := make([]Chart, len(results))
	for i, r := range results {
		charts[i] = Chart{
			Name:        r.Name,
			Version:     r.Version,
			Description: r.Description,
		}
	}
	
	return charts, nil
}

type ChartVersion struct {
	Version     string
	AppVersion  string
	Description string
}

func (c *Client) GetChartVersions(chartName string) ([]ChartVersion, error) {
	cmd := exec.Command("helm", "search", "repo", chartName, "--versions", "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("helm search versions failed: %w", err)
	}
	
	var results []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		AppVersion  string `json:"app_version"`
		Description string `json:"description"`
	}
	
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, err
	}
	
	versions := make([]ChartVersion, len(results))
	for i, r := range results {
		versions[i] = ChartVersion{
			Version:     r.Version,
			AppVersion:  r.AppVersion,
			Description: r.Description,
		}
	}
	
	return versions, nil
}

func (c *Client) GetChartValues(chartName string) (string, error) {
	cmd := exec.Command("helm", "show", "values", chartName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("helm show values failed: %w", err)
	}
	return string(output), nil
}

func (c *Client) GetChartValuesByVersion(chartName, version string) (string, error) {
	cmd := exec.Command("helm", "show", "values", chartName, "--version", version)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("helm show values failed: %w", err)
	}
	return string(output), nil
}

func (c *Client) ExportValues(chartName, outputFile string) error {
	values, err := c.GetChartValues(chartName)
	if err != nil {
		return err
	}
	
	return os.WriteFile(outputFile, []byte(values), 0644)
}

func (c *Client) GenerateTemplate(chartName, valuesFile, outputPath string) error {
	releaseName := "myrelease"
	
	args := []string{"template", releaseName, chartName, "--output-dir", outputPath}
	if valuesFile != "" {
		args = append(args, "-f", valuesFile)
	}
	
	cmd := exec.Command("helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm template failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

func (c *Client) AddRepository(name, url string) error {
	cmd := exec.Command("helm", "repo", "add", name, url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm repo add failed: %w\nOutput: %s", err, string(output))
	}

	// Update repo dopo l'aggiunta
	cmd = exec.Command("helm", "repo", "update", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm repo update failed: %w", err)
	}

	return nil
}

func (c *Client) RemoveRepository(name string) error {
	cmd := exec.Command("helm", "repo", "remove", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("helm repo remove failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}
