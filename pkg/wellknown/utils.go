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

package wellknown

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	nameToResource         = map[string]*resource{}
	nameAndGroupToResource = map[schema.GroupResource]*resource{}
)

func init() {
	for i, r := range resources {
		for _, name := range r.Names {
			nameToResource[name] = &resources[i]
			nameAndGroupToResource[schema.GroupResource{Resource: name, Group: r.Group}] = &resources[i]
		}
	}
}

// CorrectGroupResource returns the corrected GroupResource and namespaced
func CorrectGroupResource(target schema.GroupResource) (gr schema.GroupResource, namespaced bool, found bool) {
	var r *resource
	if target.Group == "" {
		r, found = nameToResource[target.Resource]
	} else {
		r, found = nameAndGroupToResource[target]
	}
	if !found {
		return gr, namespaced, false
	}
	gr = schema.GroupResource{
		Group:    r.Group,
		Resource: r.Names[0],
	}
	return gr, r.Namespaced, true
}
