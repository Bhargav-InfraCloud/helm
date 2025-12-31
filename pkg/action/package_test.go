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

package action

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"

	"helm.sh/helm/v4/internal/test/ensure"
	"helm.sh/helm/v4/pkg/chart/common"
	chart "helm.sh/helm/v4/pkg/chart/v2"
)

func TestPassphraseFileFetcher(t *testing.T) {
	secret := "secret"
	directory := ensure.TempFile(t, "passphrase-file", []byte(secret))
	testPkg := NewPackage()

	fetcher, err := testPkg.passphraseFileFetcher(path.Join(directory, "passphrase-file"), nil)
	if err != nil {
		t.Fatal("Unable to create passphraseFileFetcher", err)
	}

	passphrase, err := fetcher("key")
	if err != nil {
		t.Fatal("Unable to fetch passphrase")
	}

	if string(passphrase) != secret {
		t.Errorf("Expected %s got %s", secret, string(passphrase))
	}
}

func TestPassphraseFileFetcher_WithLineBreak(t *testing.T) {
	secret := "secret"
	directory := ensure.TempFile(t, "passphrase-file", []byte(secret+"\n\n."))
	testPkg := NewPackage()

	fetcher, err := testPkg.passphraseFileFetcher(path.Join(directory, "passphrase-file"), nil)
	if err != nil {
		t.Fatal("Unable to create passphraseFileFetcher", err)
	}

	passphrase, err := fetcher("key")
	if err != nil {
		t.Fatal("Unable to fetch passphrase")
	}

	if string(passphrase) != secret {
		t.Errorf("Expected %s got %s", secret, string(passphrase))
	}
}

func TestPassphraseFileFetcher_WithInvalidStdin(t *testing.T) {
	directory := t.TempDir()
	testPkg := NewPackage()

	stdin, err := os.CreateTemp(directory, "non-existing")
	if err != nil {
		t.Fatal("Unable to create test file", err)
	}

	if _, err := testPkg.passphraseFileFetcher("-", stdin); err == nil {
		t.Error("Expected passphraseFileFetcher returning an error")
	}
}

func TestPassphraseFileFetcher_WithStdinAndMultipleFetches(t *testing.T) {
	testPkg := NewPackage()
	stdin, w, err := os.Pipe()
	if err != nil {
		t.Fatal("Unable to create pipe", err)
	}

	passphrase := "secret-from-stdin"

	go func() {
		w.Write([]byte(passphrase + "\n"))
	}()

	for range 4 {
		fetcher, err := testPkg.passphraseFileFetcher("-", stdin)
		if err != nil {
			t.Errorf("Expected passphraseFileFetcher to not return an error, but got %v", err)
		}

		pass, err := fetcher("key")
		if err != nil {
			t.Errorf("Expected passphraseFileFetcher invocation to succeed, failed with %v", err)
		}

		if string(pass) != string(passphrase) {
			t.Errorf("Expected multiple passphrase fetch to return %q, got %q", passphrase, pass)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	type args struct {
		ver string
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			"normal semver version",
			args{
				ver: "1.1.3-23658",
			},
			nil,
		},
		{
			"Pre version number starting with 0",
			args{
				ver: "1.1.3-023658",
			},
			semver.ErrSegmentStartsZero,
		},
		{
			"Invalid version number",
			args{
				ver: "1.1.3.sd.023658",
			},
			semver.ErrInvalidSemVer,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateVersion(tt.args.ver); err != nil {
				if err != tt.wantErr {
					t.Errorf("Expected {%v}, got {%v}", tt.wantErr, err)
				}

			}
		})
	}
}

func TestOverrideAllModTimesInChart(t *testing.T) {
	// Chart fields.
	//
	// Note: This is a simplified version of `chart.Chart` for the current tests, with pointers removed to avoid shared
	// state between test cases.
	type chartFields struct {
		metadata      chart.Metadata
		raw           []common.File
		templates     []common.File
		schemaModTime time.Time
		files         []common.File
		modTime       time.Time
		dependencies  []chartFields
	}

	var (
		// Modification times.
		modTime  = time.Date(2023, 5, 1, 12, 0, 0, 0, time.UTC)
		override = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

		// Common files used in charts and dependencies.
		valuesFile = common.File{
			Name:    "values.yaml",
			ModTime: modTime,
		}
		readmeFile = common.File{
			Name:    "README.md",
			ModTime: modTime,
		}
		allFiles = []common.File{
			valuesFile,
			readmeFile,
		}

		// Test dependency charts.
		dep1 = chartFields{
			raw:           allFiles,
			templates:     allFiles,
			schemaModTime: modTime,
			files:         allFiles,
			modTime:       modTime,
		}
		dep2 = chartFields{
			raw:           allFiles,
			templates:     allFiles,
			schemaModTime: modTime,
			files:         allFiles,
			modTime:       modTime,
		}

		// Test chart metadata.
		metadata = chart.Metadata{
			Name:    "mychart",
			Version: "1.0.0",
		}
	)

	// Test cases.
	tests := []struct {
		name        string
		chartFields *chartFields
		override    time.Time
	}{
		{
			name: "override chart modification times",
			chartFields: &chartFields{
				metadata:      metadata,
				dependencies:  []chartFields{},
				raw:           allFiles,
				templates:     allFiles,
				schemaModTime: modTime,
				files:         allFiles,
				modTime:       modTime,
			},
			override: override,
		},
		{
			name: "override chart and dependencies modification times",
			chartFields: &chartFields{
				metadata: metadata,
				dependencies: []chartFields{
					dep1,
					dep2,
				},
				raw:           allFiles,
				templates:     allFiles,
				schemaModTime: modTime,
				files:         allFiles,
				modTime:       modTime,
			},
			override: override,
		},
		{
			name: "no override any modification times when override time is zero",
			chartFields: &chartFields{
				metadata: metadata,
				dependencies: []chartFields{
					dep1,
					dep2,
				},
				raw:           allFiles,
				templates:     allFiles,
				schemaModTime: modTime,
				files:         allFiles,
				modTime:       modTime,
			},
			override: time.Time{},
		},
		{
			name:        "skip override when chart is nil",
			chartFields: nil,
			override:    override,
		},
	}

	// Run test cases.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testChart *chart.Chart
			var dependencies []*chart.Chart
			if tt.chartFields != nil {
				// Create files with pointers for using in `chart.Chart` instance.
				files := make([]*common.File, len(tt.chartFields.files))
				for i := range tt.chartFields.files {
					// Create a copy of file to get a unique pointer to avoid shared state between test cases.
					file := tt.chartFields.files[i]
					files[i] = &file
				}

				// Create raw files with pointers for using in `chart.Chart` instance.
				rawFiles := make([]*common.File, len(tt.chartFields.raw))
				for i := range tt.chartFields.raw {
					// Create a copy of raw file to get a unique pointer to avoid shared state between test cases.
					rawFile := tt.chartFields.raw[i]
					rawFiles[i] = &rawFile
				}

				// Create template files with pointers for using in `chart.Chart` instance.
				templateFiles := make([]*common.File, len(tt.chartFields.templates))
				for i := range tt.chartFields.templates {
					// Create a copy of template file to get a unique pointer to avoid shared state between test cases.
					templateFile := tt.chartFields.templates[i]
					templateFiles[i] = &templateFile
				}

				// Create the `chart.Chart` instance.
				testChart = &chart.Chart{
					Metadata:      &tt.chartFields.metadata,
					Files:         files,
					SchemaModTime: modTime,
					Raw:           rawFiles,
					Templates:     templateFiles,
					Values: map[string]any{
						"replicaCount": 1,
					},
					ModTime: modTime,
				}

				// Create dependency charts to add to the `chart.Chart` instance.
				dependencies = make([]*chart.Chart, len(tt.chartFields.dependencies))
				for i := range tt.chartFields.dependencies {
					// Create files with pointers for using in current dependency chart.
					files := make([]*common.File, len(tt.chartFields.dependencies[i].files))
					for j := range tt.chartFields.dependencies[i].files {
						// Create a copy of file to get a unique pointer to avoid shared state between test cases.
						file := tt.chartFields.dependencies[i].files[j]
						files[j] = &file
					}

					// Create raw files with pointers for using in current dependency chart.
					rawFiles := make([]*common.File, len(tt.chartFields.dependencies[i].raw))
					for j := range tt.chartFields.dependencies[i].raw {
						// Create a copy of raw file to get a unique pointer to avoid shared state between test cases.
						rawFile := tt.chartFields.dependencies[i].raw[j]
						rawFiles[j] = &rawFile
					}

					// Create template files with pointers for using in current dependency chart.
					templateFiles := make([]*common.File, len(tt.chartFields.dependencies[i].templates))
					for j := range tt.chartFields.dependencies[i].templates {
						// Create a copy of template file to get a unique pointer to avoid shared state between test
						// cases.
						templateFile := tt.chartFields.dependencies[i].templates[j]
						templateFiles[j] = &templateFile
					}

					// Create the dependency `chart.Chart` instance.
					dependencies[i] = &chart.Chart{
						Metadata:      &tt.chartFields.dependencies[i].metadata,
						Files:         files,
						SchemaModTime: tt.chartFields.dependencies[i].schemaModTime,
						Raw:           rawFiles,
						Templates:     templateFiles,
						ModTime:       tt.chartFields.dependencies[i].modTime,
					}
				}

				// Add dependencies to the chart.
				testChart.AddDependency(dependencies...)
			}

			// Initialize Package action and override modification times in chart and its dependencies.
			packageCommand := NewPackage()
			packageCommand.ModTimeOverride = tt.override
			packageCommand.overrideAllModTimesInChart(testChart)

			// Determine expected modification time for the current test case.
			expectedModTime := modTime
			if !tt.override.IsZero() {
				expectedModTime = tt.override
			}

			// Assertions.
			if testChart != nil {
				// Assertions on chart modification times.
				assert.Equal(t, expectedModTime, testChart.ModTime)
				assert.Equal(t, expectedModTime, testChart.SchemaModTime)
				for _, file := range testChart.Files {
					assert.Equal(t, expectedModTime, file.ModTime)
				}
				for _, file := range testChart.Raw {
					assert.Equal(t, expectedModTime, file.ModTime)
				}
				for _, file := range testChart.Templates {
					assert.Equal(t, expectedModTime, file.ModTime)
				}

				// Assertions on chart dependencies modification times.
				for _, dep := range dependencies {
					assert.Equal(t, expectedModTime, dep.ModTime)
					assert.Equal(t, expectedModTime, dep.SchemaModTime)
					for _, file := range dep.Files {
						assert.Equal(t, expectedModTime, file.ModTime)
					}
					for _, file := range dep.Raw {
						assert.Equal(t, expectedModTime, file.ModTime)
					}
					for _, file := range dep.Templates {
						assert.Equal(t, expectedModTime, file.ModTime)
					}
				}
			}
		})
	}
}
