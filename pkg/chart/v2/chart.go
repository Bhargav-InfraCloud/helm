/*
Copyright The Helm Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/chart/common"
)

// APIVersionV1 is the API version number for version 1.
const APIVersionV1 = "v1"

// APIVersionV2 is the API version number for version 2.
const APIVersionV2 = "v2"

// aliasNameFormat defines the characters that are legal in an alias name.
var aliasNameFormat = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// Chart is a helm package that contains metadata, a default config, zero or more
// optionally parameterizable templates, and zero or more charts (dependencies).
type Chart struct {
	// Raw contains the raw contents of the files originally contained in the chart archive.
	//
	// This should not be used except in special cases like `helm show values`,
	// where we want to display the raw values, comments and all.
	Raw []*common.File `json:"-"`
	// Metadata is the contents of the Chartfile.
	Metadata *Metadata `json:"metadata"`
	// Lock is the contents of Chart.lock.
	Lock *Lock `json:"lock"`
	// Templates for this chart.
	Templates []*common.File `json:"templates"`
	// Values are default config for this chart.
	Values map[string]interface{} `json:"values"`
	// Schema is an optional JSON schema for imposing structure on Values
	Schema []byte `json:"schema"`
	// SchemaModTime the schema was last modified
	SchemaModTime time.Time `json:"schemamodtime,omitempty"`
	// Files are miscellaneous files in a chart archive,
	// e.g. README, LICENSE, etc.
	Files []*common.File `json:"files"`
	// ModTime the chart metadata was last modified
	ModTime time.Time `json:"modtime,omitzero"`

	parent       *Chart
	dependencies []*Chart
}

type CRD struct {
	// Name is the File.Name for the crd file
	Name string
	// Filename is the File obj Name including (sub-)chart.ChartFullPath
	Filename string
	// File is the File obj for the crd
	File *common.File
}

// SetDependencies replaces the chart dependencies.
func (ch *Chart) SetDependencies(charts ...*Chart) {
	ch.dependencies = nil
	ch.AddDependency(charts...)
}

// Name returns the name of the chart.
func (ch *Chart) Name() string {
	if ch.Metadata == nil {
		return ""
	}
	return ch.Metadata.Name
}

// AddDependency determines if the chart is a subchart.
func (ch *Chart) AddDependency(charts ...*Chart) {
	for i, x := range charts {
		charts[i].parent = ch
		ch.dependencies = append(ch.dependencies, x)
	}
}

// Root finds the root chart.
func (ch *Chart) Root() *Chart {
	if ch.IsRoot() {
		return ch
	}
	return ch.Parent().Root()
}

// Dependencies are the charts that this chart depends on.
func (ch *Chart) Dependencies() []*Chart { return ch.dependencies }

// IsRoot determines if the chart is the root chart.
func (ch *Chart) IsRoot() bool { return ch.parent == nil }

// Parent returns a subchart's parent chart.
func (ch *Chart) Parent() *Chart { return ch.parent }

// ChartPath returns the full path to this chart in dot notation.
func (ch *Chart) ChartPath() string {
	if !ch.IsRoot() {
		return ch.Parent().ChartPath() + "." + ch.Name()
	}
	return ch.Name()
}

// ChartFullPath returns the full path to this chart.
// Note that the path may not correspond to the path where the file can be found on the file system if the path
// points to an aliased subchart.
func (ch *Chart) ChartFullPath() string {
	if !ch.IsRoot() {
		return ch.Parent().ChartFullPath() + "/charts/" + ch.Name()
	}
	return ch.Name()
}

// Validate validates the metadata.
func (ch *Chart) Validate() error {
	return ch.Metadata.Validate()
}

// AppVersion returns the appversion of the chart.
func (ch *Chart) AppVersion() string {
	if ch.Metadata == nil {
		return ""
	}
	return ch.Metadata.AppVersion
}

// CRDs returns a list of File objects in the 'crds/' directory of a Helm chart.
// Deprecated: use CRDObjects()
func (ch *Chart) CRDs() []*common.File {
	files := []*common.File{}
	// Find all resources in the crds/ directory
	for _, f := range ch.Files {
		if strings.HasPrefix(f.Name, "crds/") && hasManifestExtension(f.Name) {
			files = append(files, f)
		}
	}
	// Get CRDs from dependencies, too.
	for _, dep := range ch.Dependencies() {
		files = append(files, dep.CRDs()...)
	}
	return files
}

// CRDObjects returns a list of CRD objects in the 'crds/' directory of a Helm chart & subcharts
func (ch *Chart) CRDObjects() []CRD {
	crds := []CRD{}
	// Find all resources in the crds/ directory
	for _, f := range ch.Files {
		if strings.HasPrefix(f.Name, "crds/") && hasManifestExtension(f.Name) {
			mycrd := CRD{Name: f.Name, Filename: filepath.Join(ch.ChartFullPath(), f.Name), File: f}
			crds = append(crds, mycrd)
		}
	}
	// Get CRDs from dependencies, too.
	for _, dep := range ch.Dependencies() {
		crds = append(crds, dep.CRDObjects()...)
	}
	return crds
}

// // DeepCopy creates a deep copy of the chart.
// func (ch *Chart) DeepCopy() *Chart {
// 	// If the chart is nil, return nil.
// 	if ch == nil {
// 		return nil
// 	}

// 	// Create a shallow copy of the chart.
// 	//
// 	// This is enough to copy the following fields:
// 	// - SchemaModTime (time.Time)
// 	// - ModTime (time.Time)
// 	//
// 	// For the other fields, i.e., Metadata, Lock, Raw, Templates, Files, Schema, Values, dependencies, and parent, a
// 	// deep copy is needed.
// 	newChart := *ch

// 	// Deep copy Metadata (pointer to struct Metadata).
// 	if ch.Metadata != nil {
// 		metadata := *ch.Metadata
// 		newChart.Metadata = &metadata
// 	}

// 	// Deep copy Lock (pointer to struct Lock).
// 	if ch.Lock != nil {
// 		newChart.Lock = ch.Lock.DeepCopy()
// 	}

// 	// Deep copy raw files (slice of pointers to struct File).
// 	if ch.Raw != nil {
// 		newChart.Raw = make([]*common.File, len(ch.Raw))
// 		for i, rawFile := range ch.Raw {
// 			if rawFile != nil {
// 				newChart.Raw[i] = rawFile.DeepCopy()
// 			}
// 		}
// 	}

// 	// Deep copy template files (slice of pointers to struct File).
// 	if ch.Templates != nil {
// 		newChart.Templates = make([]*common.File, len(ch.Templates))
// 		for i, templateFile := range ch.Templates {
// 			if templateFile != nil {
// 				newChart.Templates[i] = templateFile.DeepCopy()
// 			}
// 		}
// 	}

// 	// Deep copy files (slice of pointers to struct File).
// 	if ch.Files != nil {
// 		newChart.Files = make([]*common.File, len(ch.Files))
// 		for i, file := range ch.Files {
// 			if file != nil {
// 				newChart.Files[i] = file.DeepCopy()
// 			}
// 		}
// 	}

// 	// Deep copy Schema (byte slice).
// 	if ch.Schema != nil {
// 		// Clone the Data (byte slice) to avoid sharing underlying array.
// 		newChart.Schema = bytes.Clone(ch.Schema)
// 	}

// 	// Deep copy Values (map[string]any).
// 	newChart.Values = deepCopyValues(ch.Values)

// 	// Deep copy dependencies (slice of pointers to struct Chart).
// 	if ch.dependencies != nil {
// 		newChart.dependencies = make([]*Chart, len(ch.dependencies))
// 		for i, depChart := range ch.dependencies {
// 			newChart.dependencies[i] = depChart.DeepCopy()

// 			// Set the parent of the dependency chart to the new chart, i.e., to the deep-copied chart.
// 			if newChart.dependencies[i] != nil {
// 				newChart.dependencies[i].parent = &newChart
// 			}
// 		}
// 	}

// 	// Deep copy parent (pointer to struct Chart).
// 	if ch.parent != nil {
// 		newChart.parent = ch.parent.DeepCopy()
// 	}

// 	return &newChart
// }

// // deepCopyValues creates a deep copy of a map[string]any.
// func deepCopyValues(valuesMap map[string]any) map[string]any {
// 	// If the input map is nil, return nil.
// 	if valuesMap == nil {
// 		return nil
// 	}

// 	// Create a new map with the same length.
// 	copiedValuesMap := make(map[string]any, len(valuesMap))

// 	// Deep copy each key-value pair.
// 	for k, v := range valuesMap {
// 		copiedValuesMap[k] = deepCopyValue(v)
// 	}

// 	return copiedValuesMap
// }

// // deepCopyValue creates a deep copy of any value.
// func deepCopyValue(value any) any {
// 	switch valueType := value.(type) {
// 	case map[string]any:
// 		// For maps, recursively call deepCopyValues to deep copy it.
// 		return deepCopyValues(valueType)
// 	case []any:
// 		// For slices, deep copy each element.
// 		copiedSlice := make([]any, len(valueType))
// 		for i := range valueType {
// 			copiedSlice[i] = deepCopyValue(valueType[i])
// 		}

// 		return copiedSlice
// 	default:
// 		// For primitive types, return the value as is.
// 		return valueType
// 	}
// }

func hasManifestExtension(fname string) bool {
	ext := filepath.Ext(fname)
	return strings.EqualFold(ext, ".yaml") || strings.EqualFold(ext, ".yml") || strings.EqualFold(ext, ".json")
}
