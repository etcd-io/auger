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

	clientv3 "go.etcd.io/etcd/client/v3"
)

func (c *client) Delete(ctx context.Context, prefix string, opOpts ...OpOption) error {
	opt := opOption(opOpts)
	prefix, _, err := c.getPrefix(prefix, opt)
	if err != nil {
		return err
	}

	opts := []clientv3.OpOption{}

	if opt.name == "" {
		opts = append(opts, clientv3.WithPrefix())
	}

	if opt.response != nil {
		if opt.keysOnly {
			opts = append(opts, clientv3.WithKeysOnly())
		}
		opts = append(opts, clientv3.WithPrevKV())
	}

	resp, err := c.client.Delete(ctx, prefix, opts...)
	if err != nil {
		return err
	}

	if opt.response != nil {
		for _, kv := range resp.PrevKvs {
			r := &KeyValue{
				Key:       kv.Key,
				PrevValue: kv.Value,
			}
			err = opt.response(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
