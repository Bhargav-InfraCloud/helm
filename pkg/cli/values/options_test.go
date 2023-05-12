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
	"fmt"
	"reflect"
	"testing"

	"helm.sh/helm/v3/pkg/getter"
)

func Test_mergeMaps(t *testing.T) {
	nestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool": "stuff",
		},
	}
	anotherNestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	flatMap := map[string]interface{}{
		"foo": "bar",
		"baz": "stuff",
	}
	anotherFlatMap := map[string]interface{}{
		"testing": "fun",
	}

	testMap := mergeMaps(flatMap, nestedMap)
	equal := reflect.DeepEqual(testMap, nestedMap)
	if !equal {
		t.Errorf("Expected a nested map to overwrite a flat value. Expected: %v, got %v", nestedMap, testMap)
	}

	testMap = mergeMaps(nestedMap, flatMap)
	equal = reflect.DeepEqual(testMap, flatMap)
	if !equal {
		t.Errorf("Expected a flat value to overwrite a map. Expected: %v, got %v", flatMap, testMap)
	}

	testMap = mergeMaps(nestedMap, anotherNestedMap)
	equal = reflect.DeepEqual(testMap, anotherNestedMap)
	if !equal {
		t.Errorf("Expected a nested map to overwrite another nested map. Expected: %v, got %v", anotherNestedMap, testMap)
	}

	testMap = mergeMaps(anotherFlatMap, anotherNestedMap)
	expectedMap := map[string]interface{}{
		"testing": "fun",
		"foo":     "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	equal = reflect.DeepEqual(testMap, expectedMap)
	if !equal {
		t.Errorf("Expected a map with different keys to merge properly with another map. Expected: %v, got %v", expectedMap, testMap)
	}
}

func TestReadFile(t *testing.T) {
	var p getter.Providers
	filePath := "%a.txt"
	_, err := readFile(filePath, p)
	if err == nil {
		t.Errorf("Expected error when has special strings")
	}
}

func TestOptions_MergeValues(t *testing.T) {
	const (
		appNameKey       = `appName`
		versionKey       = `version`
		backendStackKey  = `backendStack`
		frontendStackKey = `frontendStack`
		environmentKey   = `environment`
		nodesKey         = `nodes`
		adminKey         = `admin`
		replicasKey      = `replicas`
		areaKey          = `area`
		testFlagKey      = `testFlag`
		readKey          = `read`
		writeKey         = `write`
		minKey           = `min`
		maxKey           = `max`

		testAppNameVal    = `test-app`
		testAppBENameVal  = `test-app-be`
		testAppFENameVal  = `test-app-fe`
		goVal             = `Go`
		helmVal           = `Helm`
		reactVal          = `React`
		flutterVal        = `Flutter`
		version0Val       = `0.0.0`
		version1Val       = `1.0.0`
		devEnvironmentVal = `dev`
		noOfNodesVal      = `6`
		adminVal          = `Luffy`
		replicasVal       = `3`
		financeAreaVal    = `finance`
		testFlagVal       = `test-app`
		minVal            = `100`
		maxVal            = `120`
	)

	type args struct {
		p getter.Providers
	}
	tests := []struct {
		name    string
		opts    Options
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Empty-Values",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "Values-Files",
			opts: Options{
				ValueFiles: []string{
					"testdata/noconflicts/values.yaml",
					"testdata/noconflicts/extras.yaml",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				appNameKey: testAppNameVal,
				versionKey: version0Val,
			},
			wantErr: false,
		},
		{
			name: "Values-Directories",
			opts: Options{
				ValueFiles: []string{
					"testdata/noconflicts/values.d",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				backendStackKey: []interface{}{
					goVal,
					helmVal,
				},
				frontendStackKey: []interface{}{
					reactVal,
					flutterVal,
				},
			},
			wantErr: false,
		},
		{
			name: "Values-Directories-Recursive-Read",
			opts: Options{
				ValueFiles: []string{
					"testdata/multilevelvaluesd/values.d",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				appNameKey: testAppNameVal,
				versionKey: version0Val,
				backendStackKey: []interface{}{
					goVal,
					helmVal,
				},
				frontendStackKey: []interface{}{
					reactVal,
					flutterVal,
				},
			},
			wantErr: false,
		},
		{
			name: "Values-Using-Set-String",
			opts: Options{
				ValueFiles: []string{},
				StringValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", adminKey, adminVal, replicasKey, replicasVal),
				},
				Values:     []string{},
				FileValues: []string{},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				adminKey:    adminVal,
				replicasKey: replicasVal,
			},
			wantErr: false,
		},
		{
			name: "Values-Using-Set",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values: []string{
					fmt.Sprintf("%s=%s,%s=%s", areaKey, financeAreaVal, testFlagKey, testFlagVal),
				},
				FileValues: []string{},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				areaKey:     financeAreaVal,
				testFlagKey: testFlagVal,
			},
			wantErr: false,
		},
		{
			name: "Values-Using-Set-File",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", environmentKey, "testdata/noconflicts/environment",
						nodesKey, "testdata/noconflicts/nodes"),
				},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				environmentKey: devEnvironmentVal,
				nodesKey:       noOfNodesVal,
			},
			wantErr: false,
		},
		{
			name: "Values-Using-Set-JSON",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues: []string{
					fmt.Sprintf("%s={%q:%q,%q:%q},%s={%q:%q,%q:%q}", readKey, minKey,
						minVal, maxKey, maxVal, writeKey, minKey, minVal, maxKey, maxVal),
				},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				readKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
				writeKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
			},
			wantErr: false,
		},
		{
			name: "All-Types-of-Inputs",
			opts: Options{
				ValueFiles: []string{
					"testdata/noconflicts/values.yaml",
					"testdata/noconflicts/extras.yaml",
					"testdata/noconflicts/values.d",
				},
				StringValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", adminKey, adminVal, replicasKey, replicasVal),
				},
				Values: []string{
					fmt.Sprintf("%s=%s,%s=%s", areaKey, financeAreaVal, testFlagKey, testFlagVal),
				},
				FileValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", environmentKey, "testdata/noconflicts/environment",
						nodesKey, "testdata/noconflicts/nodes"),
				},
				JSONValues: []string{
					fmt.Sprintf("%s={%q:%q,%q:%q},%s={%q:%q,%q:%q}", readKey, minKey,
						minVal, maxKey, maxVal, writeKey, minKey, minVal, maxKey, maxVal),
				},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				readKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
				writeKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
				environmentKey: devEnvironmentVal,
				nodesKey:       noOfNodesVal,
				areaKey:        financeAreaVal,
				testFlagKey:    testFlagVal,
				adminKey:       adminVal,
				replicasKey:    replicasVal,
				backendStackKey: []interface{}{
					goVal,
					helmVal,
				},
				frontendStackKey: []interface{}{
					reactVal,
					flutterVal,
				},
				appNameKey: testAppNameVal,
				versionKey: version0Val,
			},
			wantErr: false,
		},
		{
			name: "All-Types-of-Inputs-Overwritten-Values",
			opts: Options{
				ValueFiles: []string{
					"testdata/withconflicts/values.d",
					"testdata/withconflicts/values.yaml",
					"testdata/withconflicts/extras.yaml",
				},
				StringValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", adminKey, adminVal, replicasKey, replicasVal),
				},
				Values: []string{
					fmt.Sprintf("%s=%s,%s=%s", areaKey, financeAreaVal, testFlagKey, testFlagVal),
				},
				FileValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", environmentKey, "testdata/noconflicts/environment",
						nodesKey, "testdata/noconflicts/nodes"),
				},
				JSONValues: []string{
					fmt.Sprintf("%s={%q:%q,%q:%q},%s={%q:%q,%q:%q}", readKey, minKey,
						minVal, maxKey, maxVal, writeKey, minKey, minVal, maxKey, maxVal),
				},
			},
			args: args{
				p: []getter.Provider{},
			},
			want: map[string]interface{}{
				readKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
				writeKey: map[string]interface{}{
					minKey: minVal,
					maxKey: maxVal,
				},
				environmentKey: devEnvironmentVal,
				nodesKey:       noOfNodesVal,
				areaKey:        financeAreaVal,
				testFlagKey:    testFlagVal,
				adminKey:       adminVal,
				replicasKey:    replicasVal,
				backendStackKey: []interface{}{
					goVal,
					helmVal,
				},
				frontendStackKey: []interface{}{
					reactVal,
					flutterVal,
				},
				appNameKey: testAppNameVal,
				versionKey: version1Val,
			},
			wantErr: false,
		},
		{
			name: "Failure-Values-File-Missing",
			opts: Options{
				ValueFiles: []string{
					"testdata/noconflicts/values-non-existing.yaml",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Values-Directory-Missing",
			opts: Options{
				ValueFiles: []string{
					"testdata/noconflicts/values-non-existing.d",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-YAML-in-Values-File",
			opts: Options{
				ValueFiles: []string{
					"testdata/malformed/values.yaml",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-YAML-in-Values-Directory",
			opts: Options{
				ValueFiles: []string{
					"testdata/malformed/values.d",
				},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues:   []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-JSON",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues:   []string{},
				JSONValues: []string{
					fmt.Sprintf("%s={%q:%q,%q:%q,%s={%q:%q,%q:%q}", readKey, minKey,
						minVal, maxKey, maxVal, writeKey, minKey, minVal, maxKey, maxVal),
				},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-Set-Input",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values: []string{
					fmt.Sprintf("%s:%s,%s=%s", areaKey, financeAreaVal, testFlagKey, testFlagVal),
				},
				FileValues: []string{},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-String-Input",
			opts: Options{
				ValueFiles: []string{},
				StringValues: []string{
					fmt.Sprintf("%s:%s,%s=%s", adminKey, adminVal, replicasKey, replicasVal),
				},
				Values:     []string{},
				FileValues: []string{},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Set-File-Missing",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues: []string{
					fmt.Sprintf("%s=%s,%s=%s", environmentKey, "testdata/noconflicts/environment-non-existing",
						nodesKey, "testdata/noconflicts/nodes"),
				},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
		{
			name: "Failure-Malformed-File-String-Input",
			opts: Options{
				ValueFiles:   []string{},
				StringValues: []string{},
				Values:       []string{},
				FileValues: []string{
					fmt.Sprintf("%s:%s,%s=%s", environmentKey, "testdata/noconflicts/environment",
						nodesKey, "testdata/noconflicts/nodes"),
				},
				JSONValues: []string{},
			},
			args: args{
				p: []getter.Provider{},
			},
			want:    map[string]interface{}(nil),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.opts.MergeValues(tt.args.p)

			if (err != nil) != tt.wantErr {
				t.Errorf("Options.MergeValues() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected result from MergeValues() = %v, got %v", tt.want, got)
			}
		})
	}
}
