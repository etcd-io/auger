/*
Copyright 2024 The Kubernetes Authors.

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

package client

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// specialDefaultResourcePrefixes are prefixes compiled into Kubernetes.
// see https://github.com/kubernetes/kubernetes/blob/a2106b5f73fe9352f7bc0520788855d57fc301e1/pkg/kubeapiserver/default_storage_factory_builder.go#L42-L50
var specialDefaultResourcePrefixes = map[schema.GroupResource]string{
	{Group: "", Resource: "replicationcontrollers"}:     "controllers",
	{Group: "", Resource: "endpoints"}:                  "services/endpoints",
	{Group: "", Resource: "nodes"}:                      "minions",
	{Group: "", Resource: "services"}:                   "services/specs",
	{Group: "extensions", Resource: "ingresses"}:        "ingress",
	{Group: "networking.k8s.io", Resource: "ingresses"}: "ingress",
}

var specialDefaultMediaTypes = map[string]struct{}{
	"apiextensions.k8s.io":   {},
	"apiregistration.k8s.io": {},
}

// prefixFromGR returns the prefix of the given GroupResource.
func prefixFromGR(gr schema.GroupResource) (string, error) {
	if gr.Resource == "" {
		return "", fmt.Errorf("resource is empty")
	}

	if prefix, ok := specialDefaultResourcePrefixes[gr]; ok {
		return prefix, nil
	}

	if _, ok := specialDefaultMediaTypes[gr.Group]; ok {
		return gr.Group + "/" + gr.Resource, nil
	}

	if gr.Group == "" {
		return gr.Resource, nil
	}

	if !strings.Contains(gr.Group, ".") {
		return gr.Resource, nil
	}

	// TODO: This can cause errors if custom resources use this group.
	if strings.HasSuffix(gr.Group, ".k8s.io") {
		return gr.Resource, nil
	}

	// custom resources
	return gr.Group + "/" + gr.Resource, nil
}

// getPrefix returns path and wantSingle
// the path means the key based on schema.GroupResource and the resource name and namespace.
// the single means that the name is specified, this is a single resource
func getPrefix(prefix string, gr schema.GroupResource, name, namespace string) (path string, single bool, err error) {
	var arr [4]string
	s := arr[:0]
	s = append(s, prefix)

	if gr.Empty() {
		if namespace != "" || name != "" {
			return "", false, fmt.Errorf("namespace and name must be omitted if there is no GroupResource")
		}
	} else {
		p, err := prefixFromGR(gr)
		if err != nil {
			return "", false, err
		}
		s = append(s, p)
		if namespace != "" {
			s = append(s, namespace)
		}
		if name != "" {
			s = append(s, name)
			single = true
		}
	}

	if !single {
		s = append(s, "")
	}
	return strings.Join(s, "/"), single, nil
}
