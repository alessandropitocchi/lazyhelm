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

// SearchResponse represents the response from Artifact Hub search API
type SearchResponse struct {
	Packages []Package `json:"packages"`
}

// Package represents a Helm package in Artifact Hub
type Package struct {
	PackageID        string              `json:"package_id"`
	Name             string              `json:"name"`
	NormalizedName   string              `json:"normalized_name"`
	Description      string              `json:"description"`
	Version          string              `json:"version"`
	AppVersion       string              `json:"app_version"`
	Deprecated       bool                `json:"deprecated"`
	Stars            int                 `json:"stars"`
	Signed           bool                `json:"signed"`
	Signatures       []string            `json:"signatures"`
	SecurityReport   SecurityReport      `json:"security_report_summary"`
	Repository       Repository          `json:"repository"`
	Keywords         []string            `json:"keywords"`
	HomeURL          string              `json:"home_url"`
	Readme           string              `json:"readme"`
	AvailableVersions []AvailableVersion `json:"available_versions"`
	ValuesSchema     interface{}         `json:"values_schema"`
	DefaultValues    string              `json:"default_values"`
}

// Repository represents a Helm repository in Artifact Hub
type Repository struct {
	RepositoryID        string `json:"repository_id"`
	Name                string `json:"name"`
	DisplayName         string `json:"display_name"`
	URL                 string `json:"url"`
	Kind                int    `json:"kind"` // 0 = Helm
	VerifiedPublisher   bool   `json:"verified_publisher"`
	Official            bool   `json:"official"`
	OrganizationName    string `json:"organization_name"`
	OrganizationDisplay string `json:"organization_display_name"`
}

// SecurityReport represents security vulnerability summary
type SecurityReport struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Unknown  int `json:"unknown"`
}

// AvailableVersion represents an available version of a package
type AvailableVersion struct {
	Version    string `json:"version"`
	CreatedAt  int64  `json:"ts"`
	ContainsSecurityUpdates bool `json:"contains_security_updates"`
	Prerelease bool   `json:"prerelease"`
}

// GetSecurityBadge returns a colored badge based on severity
func (s SecurityReport) GetSecurityBadge() string {
	if s.Critical > 0 {
		return "🔴 Critical"
	}
	if s.High > 0 {
		return "🟠 High"
	}
	if s.Medium > 0 {
		return "🟡 Medium"
	}
	if s.Low > 0 {
		return "🟢 Low"
	}
	return "✅ Secure"
}

// GetBadges returns a string with all applicable badges
func (p Package) GetBadges() string {
	badges := ""
	if p.Repository.VerifiedPublisher {
		badges += "✓ "
	}
	if p.Signed {
		badges += "🔒 "
	}
	if p.Repository.Official {
		badges += "⭐ "
	}
	return badges
}
