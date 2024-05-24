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
	"context"
	"strings"

	"github.com/etcd-io/auger/pkg/encoding"
	"k8s.io/apimachinery/pkg/runtime/schema"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client is an interface that defines the operations that can be performed on an etcd client.
type Client interface {
	// Get is a method that retrieves a key-value pair from the etcd server.
	// It returns the revision of the key-value pair
	Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error)

	// Watch is a method that watches for changes to a key-value pair on the etcd server.
	Watch(ctx context.Context, prefix string, opOpts ...OpOption) error

	// Delete is a method that deletes a key-value pair from the etcd server.
	Delete(ctx context.Context, prefix string, opOpts ...OpOption) error

	// Put is a method that sets a key-value pair on the etcd server.
	Put(ctx context.Context, prefix string, value []byte, opOpts ...OpOption) error
}

// client is the etcd client.
type client struct {
	client *clientv3.Client
}

type Config = clientv3.Config

// NewClient creates a new etcd client.
func NewClient(conf Config) (Client, error) {
	cli, err := clientv3.New(conf)
	if err != nil {
		return nil, err
	}
	return &client{
		client: cli,
	}, nil
}

func (c *client) getPrefix(prefix string, opt Op) (string, bool, error) {
	var single bool
	var arr [4]string
	s := arr[:0]
	s = append(s, prefix)

	if !opt.gr.Empty() {
		p, err := PrefixFromGR(opt.gr)
		if err != nil {
			return "", false, err
		}
		s = append(s, p)
		if opt.namespace != "" {
			s = append(s, opt.namespace)
		}
		if opt.name != "" {
			s = append(s, opt.name)
			single = true
		}
	}
	return strings.Join(s, "/"), single, nil
}

// Op is the option for the operation.
type Op struct {
	gr        schema.GroupResource
	name      string
	namespace string
	response  func(kv *KeyValue) error
	pageLimit int64
	keysOnly  bool
	revision  int64
}

// OpOption is the option for the operation.
type OpOption func(*Op)

// WithGR sets the gr for the target.
func WithGR(gr schema.GroupResource) OpOption {
	return func(o *Op) {
		o.gr = gr
	}
}

// WithName sets the name and namespace for the target.
func WithName(name, namespace string) OpOption {
	return func(o *Op) {
		o.name = name
		o.namespace = namespace
	}
}

// WithResponse sets the response callback for the target.
func WithResponse(response func(kv *KeyValue) error) OpOption {
	return func(o *Op) {
		o.response = response
	}
}

// WithPageLimit sets the page limit for the target.
func WithPageLimit(pageLimit int64) OpOption {
	return func(o *Op) {
		o.pageLimit = pageLimit
	}
}

// WithKeysOnly sets the keys only for the target.
func WithKeysOnly() OpOption {
	return func(o *Op) {
		o.keysOnly = true
	}
}

// WithRevision sets the revision for the target.
func WithRevision(revision int64) OpOption {
	return func(o *Op) {
		o.revision = revision
	}
}

func opOption(opts []OpOption) Op {
	var opt Op
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// KeyValue is the key-value pair.
type KeyValue struct {
	Key       []byte
	Value     []byte
	PrevValue []byte
}

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

// PrefixFromGR returns the prefix of the given GroupResource.
func PrefixFromGR(gr schema.GroupResource) (string, error) {
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

// MediaTypeFromGR returns the media type of the given GroupResource.
func MediaTypeFromGR(gr schema.GroupResource) (mediaType string, err error) {
	if _, ok := specialDefaultMediaTypes[gr.Group]; ok {
		return encoding.JsonMediaType, nil
	}

	if !strings.Contains(gr.Group, ".") {
		return encoding.StorageBinaryMediaType, nil
	}

	// TODO: This can cause errors if custom resources use this group.
	if strings.HasSuffix(gr.Group, ".k8s.io") {
		return encoding.StorageBinaryMediaType, nil
	}

	return encoding.JsonMediaType, nil
}
