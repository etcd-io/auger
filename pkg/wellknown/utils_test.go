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
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCorrectGroupResource(t *testing.T) {
	type args struct {
		target schema.GroupResource
	}
	tests := []struct {
		name           string
		args           args
		wantGr         schema.GroupResource
		wantNamespaced bool
		wantFound      bool
	}{
		{
			name: "empty",
			args: args{
				target: schema.GroupResource{
					Resource: "",
				},
			},
			wantFound: false,
		},
		{
			name: "pods",
			args: args{
				target: schema.GroupResource{
					Resource: "pods",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "pods",
			},
			wantNamespaced: true,
			wantFound:      true,
		},
		{
			name: "pod",
			args: args{
				target: schema.GroupResource{
					Resource: "pod",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "pods",
			},
			wantNamespaced: true,
			wantFound:      true,
		},
		{
			name: "po",
			args: args{
				target: schema.GroupResource{
					Resource: "po",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "pods",
			},
			wantNamespaced: true,
			wantFound:      true,
		},
		{
			name: "role",
			args: args{
				target: schema.GroupResource{
					Resource: "role",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "roles",
				Group:    "rbac.authorization.k8s.io",
			},
			wantNamespaced: true,
			wantFound:      true,
		},
		{
			name: "no",
			args: args{
				target: schema.GroupResource{
					Resource: "no",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "nodes",
				Group:    "",
			},
			wantNamespaced: false,
			wantFound:      true,
		},
		{
			name: "deploy.apps",
			args: args{
				target: schema.GroupResource{
					Resource: "deploy",
					Group:    "apps",
				},
			},
			wantGr: schema.GroupResource{
				Resource: "deployments",
				Group:    "apps",
			},
			wantNamespaced: true,
			wantFound:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGr, gotNamespaced, gotFound := CorrectGroupResource(tt.args.target)
			if !reflect.DeepEqual(gotGr, tt.wantGr) {
				t.Errorf("CorrectGroupResource() gotGr = %v, want %v", gotGr, tt.wantGr)
			}
			if gotNamespaced != tt.wantNamespaced {
				t.Errorf("CorrectGroupResource() gotNamespaced = %v, want %v", gotNamespaced, tt.wantNamespaced)
			}
			if gotFound != tt.wantFound {
				t.Errorf("CorrectGroupResource() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}
