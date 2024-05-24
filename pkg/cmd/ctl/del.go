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
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type delFlagpole struct {
	Namespace string
	Output    string
	Prefix    string
}

func newCtlDelCommand() *cobra.Command {
	flags := &delFlagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "del [resource] [name]",
		Short: "Deletes the resource of k8s in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			etcdclient, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			err = delCommand(cmd.Context(), etcdclient, flags, args)

			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "key", "output format. One of: (key, none).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix to prepend to the resource")
	return cmd
}

func delCommand(ctx context.Context, etcdclient client.Client, flags *delFlagpole, args []string) error {
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
	if flags.Output == "key" {
		response = func(kv *client.KeyValue) error {
			count++
			fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			return nil
		}
	}

	opOpts := []client.OpOption{
		client.WithName(targetName, targetNamespace),
		client.WithGR(targetGr),
	}

	if response != nil {
		opOpts = append(opOpts,
			client.WithKeysOnly(),
			client.WithResponse(response),
		)
	}

	err := etcdclient.Delete(ctx, flags.Prefix,
		opOpts...,
	)
	if err != nil {
		return err
	}

	if flags.Output == "key" {
		fmt.Fprintf(os.Stderr, "delete %d keys\n", count)
	}
	return nil
}
