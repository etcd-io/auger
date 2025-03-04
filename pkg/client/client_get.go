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
	"errors"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func (c *client) Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error) {
	if prefix == "" {
		return 0, errors.New("prefix is required")
	}

	opt := opOption(opOpts)
	if opt.response == nil {
		return 0, errors.New("response is required")
	}

	path, single, err := getPrefix(prefix, opt.gr, opt.name, opt.namespace)
	if err != nil {
		return 0, err
	}

	opts := make([]clientv3.OpOption, 0, 3)

	// specify whether it is a key or a prefix
	if !single {
		opts = append(opts, clientv3.WithPrefix())
	}

	// specify an explicit revision and always use it
	if opt.revision != 0 {
		rev = opt.revision
		opts = append(opts, clientv3.WithRev(rev))
	}

	// it is a key or it is not paging
	if single || opt.chunkSize == 0 {
		resp, err := c.client.Get(ctx, path, opts...)
		if err != nil {
			return 0, err
		}

		err = iterateList(resp.Kvs, opt.response)
		if err != nil {
			return 0, err
		}
		return resp.Header.Revision, nil
	}

	// paging for content
	opts = append(opts, clientv3.WithLimit(opt.chunkSize))
	for key := path; ; {
		resp, err := c.client.Get(ctx, key, opts...)
		if err != nil {
			return 0, err
		}

		err = iterateList(resp.Kvs, opt.response)
		if err != nil {
			return 0, err
		}

		// if revision is not set, it is set to the revision of the first response.
		if rev == 0 {
			rev = resp.Header.Revision
			opts = append(opts, clientv3.WithRev(resp.Header.Revision))
		}

		if !resp.More {
			break
		}

		// move to next key
		key = string(append(resp.Kvs[len(resp.Kvs)-1].Key, 0))
	}

	return rev, nil
}
