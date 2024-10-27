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
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"helm.sh/helm/v3/pkg/cli/output"

	"github.com/gosuri/uitable"
	"k8s.io/apimachinery/pkg/api/meta"
	metatable "k8s.io/apimachinery/pkg/api/meta/table"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// getDeployed is the action for checking the named release's deployed resource list. It is the implementation
// of 'helm get deployed' subcommand.
//
// For eg. say there is an nginx release with just a deployment and a service, this will list as follows:
//
// TODO :: Bhargav :: Copy paste a real-time output here, with ones that has both namespaces, non-namespaced and crds.
// $ helm get deployed nginx
// NAMESPACE	NAME             	API_VERSION	AGE
// default  	services/nginx   	v1         	38s
// default  	deployments/nginx	apps/v1    	38s
type getDeployed struct {
	cfg *Configuration
}

// NewGetDeployed creates a new GetDeployed object with the input configuration.
func NewGetDeployed(cfg *Configuration) *getDeployed {
	return &getDeployed{
		cfg: cfg,
	}
}

// Run executes 'helm get deployed' against the named release.
func (g *getDeployed) Run(ctx context.Context, name string) ([]resourceElement, error) {
	// Check if cluster is reachable from the client
	if err := g.cfg.KubeClient.IsReachable(); err != nil {
		return nil, fmt.Errorf("cluster is not reachable: %w", err)
	}

	// Get the release details. The revision is set to 0 to get the latest revision of the release.
	release, err := g.cfg.releaseContent(name, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release content: %w", err)
	}

	mapper, err := g.cfg.RESTClientGetter.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to extract the REST mapper: %v", err)
	}

	// Create function to iterate over all the resources in the release manifest
	resourceList := make([]resourceElement, 0)
	listResourcesFn := kio.FilterFunc(func(resources []*yaml.RNode) ([]*yaml.RNode, error) {
		// Iterate over the resource in manifest YAML
		for _, manifest := range resources {
			// Process resource record for "helm get deployed"
			resource, err := g.processResourceRecord(manifest, mapper)
			if err != nil {
				return nil, err
			}

			resourceList = append(resourceList, *resource)
		}

		// The current command shouldn't alter the list of resources. Hence returning resources list as it.
		return resources, nil
	})

	// Run the manifest YAML through the function to process the resources list
	err = kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: strings.NewReader(release.Manifest)}},
		Filters: []kio.Filter{listResourcesFn},
	}.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to process release manifests: %w", err)
	}

	return resourceList, nil
}

func (g *getDeployed) processResourceRecord(manifest *yaml.RNode, mapper meta.RESTMapper) (*resourceElement, error) {
	manifestStr, err := manifest.String()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the string format of the manifest: %v", err)
	}

	resourceList, err := g.cfg.KubeClient.Build(bytes.NewBufferString(manifestStr), false)
	if err != nil {
		return nil, fmt.Errorf("failed to build resource list: %v", err)
	}

	list, err := g.cfg.KubeClient.Get(resourceList, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get the resource from cluster: %v", err)
	}

	metaObj, obj, err := extractObjectFromList(list, manifest.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to extract object from the output resource list: %v", err)
	}

	resourceMapping, err := restMapping(obj, mapper)
	if err != nil {
		return nil, fmt.Errorf("failed to get the REST mapping for the resource: %v", err)
	}

	return &resourceElement{
		Resource:          resourceMapping.Resource.Resource,
		Name:              manifest.GetName(),
		Namespace:         metaObj.GetNamespace(),
		APIVersion:        manifest.GetApiVersion(),
		CreationTimestamp: metaObj.GetCreationTimestamp(),
	}, nil
}

func extractObjectFromList(list map[string][]runtime.Object, name string) (metav1.Object, runtime.Object, error) {
	for _, item := range list {
		for _, obj := range item {
			metaObj, ok := obj.(metav1.Object)
			if !ok {
				return nil, nil, fmt.Errorf("object does not implement metav1.Object interface")
			}

			if metaObj.GetName() != name {
				continue
			}

			return metaObj, obj, nil
		}
	}

	return nil, nil, fmt.Errorf("object matching %q not found in the list", name)
}

func restMapping(obj runtime.Object, mapper meta.RESTMapper) (*meta.RESTMapping, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to find RESTMapping: %v", err)
	}

	return mapping, nil
}

type resourceElement struct {
	Name              string      `json:"name"`              // Resource's name
	Namespace         string      `json:"namespace"`         // Resource's namespace
	APIVersion        string      `json:"apiVersion"`        // Resource's group-version
	Resource          string      `json:"resource"`          // Resource type (eg. pods, deployments, etc.)
	CreationTimestamp metav1.Time `json:"creationTimestamp"` // Resource creation timestamp
}

type resourceListWriter struct {
	releases  []resourceElement // Resources list
	noHeaders bool              // Toggle to disable headers in tabular format
}

// NewResourceListWriter creates a output writer for Kubernetes resources to be listed with 'helm get deployed'
func NewResourceListWriter(resources []resourceElement, noHeaders bool) output.Writer {
	return &resourceListWriter{resources, noHeaders}
}

// WriteTable prints the resources list in a tabular format
func (r *resourceListWriter) WriteTable(out io.Writer) error {
	// Create table writer
	table := uitable.New()

	// Add headers if enabled
	if !r.noHeaders {
		table.AddRow("NAMESPACE", "NAME", "API_VERSION", "AGE")
	}

	// Add resources to table
	for _, r := range r.releases {
		table.AddRow(
			r.Namespace,                              // Namespace
			fmt.Sprintf("%s/%s", r.Resource, r.Name), // Name
			r.APIVersion,                             // API version
			metatable.ConvertToHumanReadableDateType(r.CreationTimestamp), // Age
		)
	}

	// Format the table and write to output writer
	return output.EncodeTable(out, table)
}

// WriteTable prints the resources list in a JSON format
func (r *resourceListWriter) WriteJSON(out io.Writer) error {
	return output.EncodeJSON(out, r.releases)
}

// WriteTable prints the resources list in a YAML format
func (r *resourceListWriter) WriteYAML(out io.Writer) error {
	return output.EncodeYAML(out, r.releases)
}
