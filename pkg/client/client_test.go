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

	"github.com/etcd-io/auger/pkg/encoding"
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
			gotPrefix, err := PrefixFromGR(tt.args.gr)
			if (err != nil) != tt.wantErr {
				t.Errorf("PrefixFromGR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPrefix != tt.wantPrefix {
				t.Errorf("PrefixFromGR() gotPrefix = %v, want %v", gotPrefix, tt.wantPrefix)
			}
		})
	}
}

func TestMediaTypeFromGR(t *testing.T) {
	type args struct {
		gr schema.GroupResource
	}
	tests := []struct {
		name          string
		args          args
		wantMediaType string
		wantErr       bool
	}{
		{
			name: "pod",
			args: args{
				gr: schema.GroupResource{
					Group:    "",
					Resource: "pods",
				},
			},
			wantMediaType: encoding.StorageBinaryMediaType,
		},
		{
			name: "deployment",
			args: args{
				gr: schema.GroupResource{
					Group:    "apps",
					Resource: "deployments",
				},
			},
			wantMediaType: encoding.StorageBinaryMediaType,
		},
		{
			name: "service",
			args: args{
				gr: schema.GroupResource{
					Group:    "",
					Resource: "services",
				},
			},
			wantMediaType: encoding.StorageBinaryMediaType,
		},
		{
			name: "ingress",
			args: args{
				gr: schema.GroupResource{
					Group:    "networking.k8s.io",
					Resource: "ingresses",
				},
			},
			wantMediaType: encoding.StorageBinaryMediaType,
		},
		{
			name: "scheduling.k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "scheduling.k8s.io",
					Resource: "priorityclasses",
				},
			},
			wantMediaType: encoding.StorageBinaryMediaType,
		},
		{
			name: "apiextensions.k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "apiextensions.k8s.io",
					Resource: "customresourcedefinitions",
				},
			},
			wantMediaType: encoding.JsonMediaType,
		},
		{
			name: "x-k8s.io",
			args: args{
				gr: schema.GroupResource{
					Group:    "auger.x-k8s.io",
					Resource: "foo",
				},
			},
			wantMediaType: encoding.JsonMediaType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMediaType, err := MediaTypeFromGR(tt.args.gr)
			if (err != nil) != tt.wantErr {
				t.Errorf("MediaTypeFromGR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMediaType != tt.wantMediaType {
				t.Errorf("MediaTypeFromGR() gotMediaType = %v, want %v", gotMediaType, tt.wantMediaType)
			}
		})
	}
}
