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

	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.etcd.io/etcd/api/v3/mvccpb"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client is an interface that defines the operations that can be performed on an etcd client.
type Client interface {
	// Get is a method that retrieves a key-value pair from the etcd server.
	// It returns the revision of the key-value pair
	Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error)

	// Watch is a method that watches for changes to a key-value pair on the etcd server.
	Watch(ctx context.Context, prefix string, opOpts ...OpOption) error
}

// client is the etcd client.
type client struct {
	client *clientv3.Client
}

// NewClient creates a new etcd client.
func NewClient(cli *clientv3.Client) Client {
	return &client{
		client: cli,
	}
}

// op is the option for the operation.
type op struct {
	gr        schema.GroupResource
	name      string
	namespace string
	// this is required if it is a query operation
	response func(kv *KeyValue) error
	// max number of results per clientv3 request.
	chunkSize int64
	revision  int64
}

// OpOption is the option for the operation.
type OpOption func(*op)

// WithGroupResource sets the schema.GroupResource for the target.
func WithGroupResource(gr schema.GroupResource) OpOption {
	return func(o *op) {
		o.gr = gr
	}
}

// WithName sets the name and namespace for the target.
func WithName(name, namespace string) OpOption {
	return func(o *op) {
		o.name = name
		o.namespace = namespace
	}
}

// WithResponse sets the response callback for the target.
func WithResponse(response func(kv *KeyValue) error) OpOption {
	return func(o *op) {
		o.response = response
	}
}

// WithChunkSize sets the max number of results per clientv3 request.
func WithChunkSize(chunkSize int64) OpOption {
	return func(o *op) {
		o.chunkSize = chunkSize
	}
}

// WithRevision sets the revision for the target.
func WithRevision(revision int64) OpOption {
	return func(o *op) {
		o.revision = revision
	}
}

func opOption(opts []OpOption) op {
	var opt op
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// KeyValue is the key-value pair.
type KeyValue struct {
	Key   []byte
	Value []byte

	PrevValue []byte
}

func iterateList(kvs []*mvccpb.KeyValue, callback func(kv *KeyValue) error) error {
	for _, kv := range kvs {
		err := callback(&KeyValue{
			Key:   kv.Key,
			Value: kv.Value,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
