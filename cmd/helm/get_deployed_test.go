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

package main

import (
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
)

func TestGetDeployed(t *testing.T) {
	var MockManifest = `apiVersion: v1
kind: Secret
metadata:
  name: fixture
  namespace: default
`

	releasesMockWithStatus := func(info *release.Info, hooks ...*release.Hook) []*release.Release {
		info.LastDeployed = helmtime.Unix(1452902400, 0).UTC()
		return []*release.Release{{
			Name:      "luffy",
			Namespace: "default",
			Info:      info,
			Chart: &chart.Chart{
				Metadata: &chart.Metadata{
					Name:       "name",
					Version:    "1.2.3",
					AppVersion: "3.2.1",
				},
				Templates: []*chart.File{
					{
						Name: "templates/secret.yaml",
						Data: []byte(MockManifest),
					},
				},
			},
			Hooks:    hooks,
			Manifest: MockManifest,
		}}
	}

	tests := []cmdTestCase{{
		name:   "get deployed with release",
		cmd:    "get deployed luffy",
		golden: "output/get-deployed.txt",
		rels: releasesMockWithStatus(&release.Info{
			Status: release.StatusDeployed,
		}),
	}}

	runTestCmd(t, tests)
}
