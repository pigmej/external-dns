/*
Copyright 2017 The Kubernetes Authors.

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

package source

import (
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/kubernetes-incubator/external-dns/endpoint"
)

// ingressSource is an implementation of Source for Kubernetes ingress objects.
// Ingress implementation will use the spec.rules.host value for the hostname
// Ingress annotations are ignored
type ingressSource struct {
	client    kubernetes.Interface
	namespace string
}

// NewIngressSource creates a new ingressSource with the given client and namespace scope.
func NewIngressSource(client kubernetes.Interface, namespace string) Source {
	return &ingressSource{client: client, namespace: namespace}
}

// Endpoints returns endpoint objects for each host-target combination that should be processed.
// Retrieves all ingress resources on all namespaces
func (sc *ingressSource) Endpoints() ([]endpoint.Endpoint, error) {
	ingresses, err := sc.client.Extensions().Ingresses(sc.namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	endpoints := []endpoint.Endpoint{}

	for _, ing := range ingresses.Items {
		ingEndpoints := endpointsFromIngress(&ing)
		endpoints = append(endpoints, ingEndpoints...)
	}

	return endpoints, nil
}

// endpointsFromIngress extracts the endpoints from ingress object
func endpointsFromIngress(ing *v1beta1.Ingress) []endpoint.Endpoint {
	var endpoints []endpoint.Endpoint

	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		for _, lb := range ing.Status.LoadBalancer.Ingress {
			if lb.IP != "" {
				endpoints = append(endpoints, endpoint.Endpoint{
					DNSName: sanitizeHostname(rule.Host),
					Target:  lb.IP,
				})
			}
			if lb.Hostname != "" {
				endpoints = append(endpoints, endpoint.Endpoint{
					DNSName: sanitizeHostname(rule.Host),
					Target:  lb.Hostname,
				})
			}
		}
	}

	return endpoints
}

// sanitizeHostname appends a trailing dot to a hostname if it's missing.
func sanitizeHostname(hostname string) string {
	return strings.Trim(hostname, ".") + "."
}
