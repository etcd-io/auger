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

package ctl

import (
	"context"
	"fmt"
	"os"

	"github.com/etcd-io/auger/pkg/client"
	"github.com/etcd-io/auger/pkg/encoding"
	"github.com/etcd-io/auger/pkg/scheme"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type getFlagpole struct {
	Namespace string
	Output    string
	ChunkSize int64
	Watch     bool
	WatchOnly bool
	Prefix    string
}

func newCtlGetCommand() *cobra.Command {
	flags := &getFlagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "get [resource] [name]",
		Short: "Gets the resource of k8s in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			etcdclient, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			err = getCommand(cmd.Context(), etcdclient, flags, args)

			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "yaml", "output format. One of: (json, yaml, raw, key).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().BoolVarP(&flags.Watch, "watch", "w", false, "after listing/getting the requested object, watch for changes")
	cmd.Flags().BoolVar(&flags.WatchOnly, "watch-only", false, "watch for changes to the requested object(s), without listing/getting first")
	cmd.Flags().Int64Var(&flags.ChunkSize, "chunk-size", 500, "chunk size of the list pager")
	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix to prepend to the resource")

	return cmd
}

func getCommand(ctx context.Context, etcdclient client.Client, flags *getFlagpole, args []string) error {
	var targetGr schema.GroupResource
	var targetName string
	var targetNamespace string
	if len(args) != 0 {
		// TODO: Support get information from CRD and scheme.Codecs
		//       Support short name
		//       Check for namespaced

		gr := schema.ParseGroupResource(args[0])
		if gr.Empty() {
			return fmt.Errorf("invalid resource %q", args[0])
		}
		targetGr = gr
		targetNamespace = flags.Namespace
		if len(args) >= 2 {
			targetName = args[1]
		}
	}

	var count int
	var response func(kv *client.KeyValue) error

	switch flags.Output {
	case "json":
		outMediaType := encoding.JsonMediaType
		response = func(kv *client.KeyValue) error {
			count++
			value := kv.Value
			if value == nil {
				value = kv.PrevValue
			}
			inMediaType, _, err := encoding.DetectAndExtract(value)
			if err != nil {
				fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
				return nil
			}
			data, _, err := encoding.Convert(scheme.Codecs, inMediaType, outMediaType, value)
			if err != nil {
				fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
			} else {
				fmt.Fprintf(os.Stdout, "---\n# %s | %s\n%s\n", kv.Key, inMediaType, data)
			}
			return nil
		}
	case "yaml":
		outMediaType := encoding.YamlMediaType
		response = func(kv *client.KeyValue) error {
			count++
			value := kv.Value
			if value == nil {
				value = kv.PrevValue
			}
			inMediaType, _, err := encoding.DetectAndExtract(value)
			if err != nil {
				fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
				return nil
			}
			data, _, err := encoding.Convert(scheme.Codecs, inMediaType, outMediaType, value)
			if err != nil {
				fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
			} else {
				fmt.Fprintf(os.Stdout, "---\n# %s | %s\n%s\n", kv.Key, inMediaType, data)
			}
			return nil
		}
	case "raw":
		response = func(kv *client.KeyValue) error {
			count++
			fmt.Fprintf(os.Stdout, "%s\n%s\n", kv.Key, kv.Value)
			return nil
		}
	case "key":
		response = func(kv *client.KeyValue) error {
			count++
			fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			return nil
		}
	default:
		return fmt.Errorf("unsupported output format: %s", flags.Output)
	}

	opOpts := []client.OpOption{
		client.WithName(targetName, targetNamespace),
		client.WithGR(targetGr),
		client.WithPageLimit(flags.ChunkSize),
		client.WithResponse(response),
	}

	if flags.Output == "key" {
		opOpts = append(opOpts,
			client.WithKeysOnly(),
		)
	}

	var err error
	if flags.Watch {
		var rev int64
		if !flags.WatchOnly {
			rev, err = etcdclient.Get(ctx, flags.Prefix,
				opOpts...,
			)
			if err != nil {
				return err
			}
		}

		opOpts = append(opOpts, client.WithRevision(rev))

		err = etcdclient.Watch(ctx, flags.Prefix,
			opOpts...,
		)
		if err != nil {
			return err
		}
	} else {
		_, err = etcdclient.Get(ctx, flags.Prefix,
			opOpts...,
		)
		if err != nil {
			return err
		}

		if flags.Output == "key" {
			fmt.Fprintf(os.Stderr, "get %d keys\n", count)
		}
	}
	return nil
}
