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
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestPrefixFromGR(t *testing.T) {
	type args struct {
		gr schema.GroupResource
	}
	tests := []struct {
		name       string
		args       args
		wantPrefix string
		wantErr    bool
	}{
		{
			name: "pod",
			args: args{
				gr: schema.GroupResource{
					Group:    "",
					Resource: "pods",
				},
			},
			wantPrefix: "pods",
			wantErr:    false,
		},
		{
			name: "deployment",
			args: args{
				gr: schema.GroupResource{
					Group:    "apps",
					Resource: "deployments",
				},
			},
			wantPrefix: "deployments",
			wantErr:    false,
		},
		{
			name: "service",
			args: args{
				gr: schema.GroupResource{
					Group:    "",
					Resource: "services",
				},
			},
			wantPrefix: "services/specs",
			wantErr:    false,
		},
		{
			name: "ingress",
			args: args{
				gr: schema.GroupResource{
					Group:    "networking.k8s.io",
					Resource: "ingresses",
				},
			},
			wantPrefix: "ingress",
		},
		{
			name: "apiextensions.k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
				},
			},
			wantPrefix: "apiextensions.k8s.io/customresourcedefinitions",
		},
		{
			name: "scheduling.k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "scheduling.k8s.io",
					Resource: "priorityclasses",
				},
			},
			wantPrefix: "priorityclasses",
		},
		{
			name: "x-k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "auger.x-k8s.io",
					Resource: "foo",
				},
			},
			wantPrefix: "auger.x-k8s.io/foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPrefix, err := prefixFromGR(tt.args.gr)
			if (err != nil) != tt.wantErr {
				t.Errorf("prefixFromGR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf("prefixFromGR() gotPrefix = %v, wantPath %v", gotPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestGetPrefix(t *testing.T) {
	type args struct {
		prefix    string
		gr        schema.GroupResource
		name      string
		namespace string
	}
	tests := []struct {
		name       string
		args       args
		wantPath   string
		wantSingle bool
		wantErr    bool
	}{
		{
			name: "all",
			args: args{
				prefix: "/registry",
			},
			wantPath:   "/registry/",
			wantSingle: false,
		},
		{
			name: "wantSingle pod",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "",
					Resource: "pods",
				},
				name:      "pod",
				namespace: "default",
			},
			wantPath:   "/registry/pods/default/pod",
			wantSingle: true,
		},
		{
			name: "pods",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "",
					Resource: "pods",
				},
				namespace: "default",
			},
			wantPath:   "/registry/pods/default/",
			wantSingle: false,
		},
		{
			name: "wantSingle node",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "",
					Resource: "nodes",
				},
				name: "node",
			},
			wantPath:   "/registry/minions/node",
			wantSingle: true,
		},
		{
			name: "nodes",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "",
					Resource: "nodes",
				},
			},
			wantPath:   "/registry/minions/",
			wantSingle: false,
		},
		{
			name: "cr",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "auger.x-k8s.io",
					Resource: "foo",
				},
			},
			wantPath:   "/registry/auger.x-k8s.io/foo/",
			wantSingle: false,
		},
		{
			name: "apiservices v1.apps",
			args: args{
				prefix: "/registry",
				gr: schema.GroupResource{
					Group:    "apiregistration.k8s.io",
					Resource: "apiservices",
				},
				name: "v1.apps",
			},
			wantPath:   "/registry/apiregistration.k8s.io/apiservices/v1.apps",
			wantSingle: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotSingle, err := getPrefix(tt.args.prefix, tt.args.gr, tt.args.name, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPrefix() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPath != tt.wantPath {
				t.Errorf("getPrefix() gotPath = %v, wantPath %v", gotPath, tt.wantPath)
			}
			if gotSingle != tt.wantSingle {
				t.Errorf("getPrefix() gotSingle = %v, wantSingle %v", gotSingle, tt.wantSingle)
			}
		})
	}
}
