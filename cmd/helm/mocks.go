package main

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"helm.sh/helm/v3/pkg/action"
)

var _ action.RESTClientGetter = (*MockRESTClientGetter)(nil)

// MockRESTClientGetter is a struct that implements the RESTClientGetter interface.
type MockRESTClientGetter struct {
	RestConfig *rest.Config
	Discovery  discovery.CachedDiscoveryInterface
	RestMapper meta.RESTMapper
}

// ToRESTConfig mocks the ToRESTConfig method.
func (m *MockRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return m.RestConfig, nil
}

// ToDiscoveryClient mocks the ToDiscoveryClient method.
func (m *MockRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return m.Discovery, nil
}

// ToRESTMapper mocks the ToRESTMapper method.
func (m *MockRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return m.RestMapper, nil
}

var _ meta.RESTMapper = (*MockRESTMapper)(nil)

type MockRESTMapper struct {
	meta.RESTMapper

	APIGroup metav1.APIGroup
}

// RESTMapping implements meta.RESTMapper.
func (m *MockRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	// Simulate returning a RESTMapping if the GroupKind matches
	if gk.Group == m.APIGroup.Name {
		return &meta.RESTMapping{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   gk.Group,
				Version: versions[0],
				Kind:    gk.Kind,
			},
			Resource: schema.GroupVersionResource{
				Group:    gk.Group,
				Version:  versions[0],
				Resource: "secrets",
			},
		}, nil
	}
	// Return error if GroupKind does not match
	return nil, errors.NewNotFound(schema.GroupResource{
		Group:    gk.Group,
		Resource: "secrets",
	}, gk.Kind)
}
