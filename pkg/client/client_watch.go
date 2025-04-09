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

func (c *client) Watch(ctx context.Context, prefix string, opOpts ...OpOption) error {
	opt := opOption(opOpts)
	if opt.response == nil {
		return fmt.Errorf("response is required")
	}

	path, single, err := getPrefix(prefix, opt.gr, opt.name, opt.namespace)
	if err != nil {
		return err
	}

	opts := make([]clientv3.OpOption, 0, 3)

	if !single {
		opts = append(opts, clientv3.WithPrefix())
	}

	if opt.revision != 0 {
		opts = append(opts, clientv3.WithRev(opt.revision))
	}

	opts = append(opts, clientv3.WithPrevKV())

	watchChan := c.client.Watch(ctx, path, opts...)
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			r := &KeyValue{
				Key:   event.Kv.Key,
				Value: event.Kv.Value,
			}
			if event.PrevKv != nil {
				r.PrevValue = event.PrevKv.Value
			}
			err := opt.response(r)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
