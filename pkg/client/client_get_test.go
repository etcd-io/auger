/*
Copyright 2026 The Kubernetes Authors.

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
	"bytes"
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// fakeKV serves Get from sorted keys the way etcd does: the range
// [key, end), at most limit keys, and More when keys were left out.
type fakeKV struct {
	clientv3.KV
	keys []string
}

func (f *fakeKV) Get(_ context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	op := clientv3.OpGet(key, opts...)
	end := op.RangeBytes()
	resp := &clientv3.GetResponse{Header: &etcdserverpb.ResponseHeader{Revision: 10}}
	for _, k := range f.keys {
		if k < key {
			continue
		}
		if len(end) == 0 {
			if k != key {
				continue
			}
		} else if bytes.Compare([]byte(k), end) >= 0 {
			continue
		}
		if op.Limit() > 0 && int64(len(resp.Kvs)) == op.Limit() {
			resp.More = true
			break
		}
		resp.Kvs = append(resp.Kvs, &mvccpb.KeyValue{Key: []byte(k)})
	}
	resp.Count = int64(len(resp.Kvs))
	return resp, nil
}

func TestGetChunkedReturnsEveryKey(t *testing.T) {
	var keys []string
	for i := 0; i < 7; i++ {
		keys = append(keys, fmt.Sprintf("/registry/pods/default/pod-%d", i))
	}
	keys = append(keys, "/registry/pods/kube-system/coredns", "/registry/secrets/default/token")

	for _, tc := range []struct {
		name      string
		chunkSize int64
		limit     int64
		want      int
	}{
		{name: "one chunk", chunkSize: 500, want: 8},
		{name: "several chunks", chunkSize: 3, want: 8},
		{name: "several chunks with a limit", chunkSize: 3, limit: 5, want: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(&clientv3.Client{KV: &fakeKV{keys: keys}})

			var got []string
			_, err := c.Get(context.Background(), "/registry",
				WithGroupResource(schema.GroupResource{Resource: "pods"}),
				WithChunkSize(tc.chunkSize),
				WithLimit(tc.limit),
				WithResponse(func(kv *KeyValue) error {
					got = append(got, string(kv.Key))
					return nil
				}),
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.want {
				t.Fatalf("got %d keys, want %d: %v", len(got), tc.want, got)
			}
			for i, k := range got {
				if k != keys[i] {
					t.Errorf("key %d is %q, want %q", i, k, keys[i])
				}
			}
		})
	}
}
