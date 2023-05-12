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

package values

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/strvals"
)

const yamlExt = `.yaml`

// Options captures the different ways to specify values
type Options struct {
	ValueFiles   []string // -f/--values
	StringValues []string // --set-string
	Values       []string // --set
	FileValues   []string // --set-file
	JSONValues   []string // --set-json
}

// MergeValues merges values from files or files in directories specified via -f/--values,
// and directly via --set-json, --set, --set-string,
// or --set-file, marshaling them to YAML
func (opts *Options) MergeValues(p getter.Providers) (map[string]interface{}, error) {
	base := map[string]interface{}{}

	// User specified a file or directory via -f/--values
	for _, filePath := range opts.ValueFiles {
		var err error

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		// Check if file is a directory
		if fileInfo.IsDir() {
			// Recursive list of YAML files in input values directory
			filesInDir, err := recursiveListOfFiles(filePath, yamlExt)
			if err != nil {
				// Error already wrapped
				return nil, err
			}
			for _, fileInDir := range filesInDir {
				base, err = mergeValuesFile(base, fileInDir, p)
				if err != nil {
					return nil, err
				}
			}
		} else {
			base, err = mergeValuesFile(base, filePath, p)
			if err != nil {
				return nil, err
			}
		}
	}

	// User specified a value via --set-json
	for _, value := range opts.JSONValues {
		if err := strvals.ParseJSON(value, base); err != nil {
			return nil, errors.Errorf("failed parsing --set-json data %s", value)
		}
	}

	// User specified a value via --set
	for _, value := range opts.Values {
		if err := strvals.ParseInto(value, base); err != nil {
			return nil, errors.Wrap(err, "failed parsing --set data")
		}
	}

	// User specified a value via --set-string
	for _, value := range opts.StringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return nil, errors.Wrap(err, "failed parsing --set-string data")
		}
	}

	// User specified a value via --set-file
	for _, value := range opts.FileValues {
		reader := func(rs []rune) (interface{}, error) {
			bytes, err := readFile(string(rs), p)
			if err != nil {
				return nil, err
			}
			return string(bytes), err
		}
		if err := strvals.ParseIntoFile(value, base, reader); err != nil {
			return nil, errors.Wrap(err, "failed parsing --set-file data")
		}
	}

	return base, nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// mergeFile load and parse a values file and merge its content with the map provided
func mergeValuesFile(base map[string]interface{}, filePath string, p getter.Providers) (map[string]interface{}, error) {
	currentMap := map[string]interface{}{}

	bytes, err := readFile(filePath, p)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", filePath)
	}
	// Merge with the previous map
	return mergeMaps(base, currentMap), err
}

// readFile load a file from stdin, the local directory, or a remote file with a url.
func readFile(filePath string, p getter.Providers) ([]byte, error) {
	if strings.TrimSpace(filePath) == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	u, err := url.Parse(filePath)
	if err != nil {
		return nil, err
	}

	// FIXME: maybe someone handle other protocols like ftp.
	g, err := p.ByScheme(u.Scheme)
	if err != nil {
		return ioutil.ReadFile(filePath)
	}
	data, err := g.Get(filePath, getter.WithURL(filePath))
	if err != nil {
		return nil, err
	}
	return data.Bytes(), err
}

// recursiveListOfFiles returns the list of filenames in input directory
// recursively in the format: [<dir>/<filename>, ..., <dir>/<sub-dir>/<filename> ...]
//
// File extension prepended with dot is required as input. Eg.: .yaml, .json, etc.
// If input file extension is empty, all files are returned. i.e., no filtering.
func recursiveListOfFiles(dir, ext string) ([]string, error) {
	var filenames []string

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s 's info", path)
			}

			// Filter files based on input extension
			if ext != "" {
				// When input file extension in not empty. i.e., list should be filtered.

				if filepath.Ext(path) != ext {
					// When file extension doesn't match input
					return nil
				}
			}

			// When file extension in input is empty.
			// When file extension is not empty and file extension matches input
			filenames = append(filenames, path)

			return nil
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to recursively list files in directory %s", dir)
	}

	return filenames, nil
}
