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
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func (c *client) Get(ctx context.Context, prefix string, opOpts ...OpOption) (rev int64, err error) {
	opt := opOption(opOpts)
	if opt.response == nil {
		return 0, fmt.Errorf("response is required")
	}

	prefix, single, err := c.getPrefix(prefix, opt)
	if err != nil {
		return 0, err
	}

	opts := []clientv3.OpOption{}
	if opt.keysOnly {
		opts = append(opts, clientv3.WithKeysOnly())
	}

	if single || opt.pageLimit == 0 {
		if !single {
			opts = append(opts, clientv3.WithPrefix())
		}
		resp, err := c.client.Get(ctx, prefix, opts...)
		if err != nil {
			return 0, err
		}
		for _, kv := range resp.Kvs {
			r := &KeyValue{
				Key:   kv.Key,
				Value: kv.Value,
			}
			err := opt.response(r)
			if err != nil {
				return 0, err
			}
		}
		return resp.Header.Revision, nil
	}

	respchan := make(chan clientv3.GetResponse, 10)
	errchan := make(chan error, 1)
	var revision int64

	go func() {
		defer close(respchan)
		defer close(errchan)

		var key string

		opts := append(opts, clientv3.WithLimit(opt.pageLimit))
		if opt.revision != 0 {
			revision = opt.revision
			opts = append(opts, clientv3.WithRev(revision))
		}

		if len(prefix) == 0 {
			// If len(s.prefix) == 0, we will sync the entire key-value space.
			// We then range from the smallest key (0x00) to the end.
			opts = append(opts, clientv3.WithFromKey())
			key = "\x00"
		} else {
			// If len(s.prefix) != 0, we will sync key-value space with given prefix.
			// We then range from the prefix to the next prefix if exists. Or we will
			// range from the prefix to the end if the next prefix does not exists.
			opts = append(opts, clientv3.WithRange(clientv3.GetPrefixRangeEnd(prefix)))
			key = prefix
		}

		for {
			resp, err := c.client.Get(ctx, key, opts...)
			if err != nil {
				errchan <- err
				return
			}

			respchan <- *resp

			if revision == 0 {
				revision = resp.Header.Revision
				opts = append(opts, clientv3.WithRev(resp.Header.Revision))
			}

			if !resp.More {
				return
			}

			// move to next key
			key = string(append(resp.Kvs[len(resp.Kvs)-1].Key, 0))
		}
	}()

	for resp := range respchan {
		for _, kv := range resp.Kvs {
			r := &KeyValue{
				Key:   kv.Key,
				Value: kv.Value,
			}
			err := opt.response(r)
			if err != nil {
				return 0, err
			}
		}
	}

	err = <-errchan
	if err != nil {
		return 0, err
	}

	return revision, nil
}
