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

package command

import (
	"context"
	"fmt"
	"os"

	"github.com/etcd-io/auger/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type getFlagpole struct {
	Namespace string
	Output    string
	ChunkSize int64
	Prefix    string

	Watch     bool
	WatchOnly bool
}

var (
	getExample = `
  # List a single service with namespace "default" and name "kubernetes"
  augerctl get services -n default kubernetes
  # Nearly equivalent
  kubectl get services -n default kubernetes -o yaml

  # List a single resource of type "priorityclasses" and name "system-node-critical" without namespaced
  augerctl get priorityclasses system-node-critical
  # Nearly equivalent
  kubectl get priorityclasses system-node-critical -A -o yaml

  # List all leases with namespace "kube-system"
  augerctl get leases -n kube-system
  # Nearly equivalent
  kubectl get leases -n kube-system -o yaml

  # Watch all leases with namespace "kube-system"
  augerctl get leases -n kube-system -w
  # Nearly equivalent
  kubectl get leases -n kube-system -w -o yaml

  # List a single resource of type "apiservices.apiregistration.k8s.io" and name "v1.apps"
  augerctl get apiservices.apiregistration.k8s.io v1.apps
  # Nearly equivalent
  kubectl get apiservices.apiregistration.k8s.io v1.apps -o yaml

  # List all resources
  augerctl get
  # Nearly equivalent
  kubectl get $(kubectl api-resources --verbs=list --output=name | paste -s -d, - ) -A -o yaml
`
)

func newCtlGetCommand(f *flagpole) *cobra.Command {
	flags := &getFlagpole{}

	cmd := &cobra.Command{
		Args:    cobra.RangeArgs(0, 2),
		Use:     "get [resource] [name]",
		Short:   "Gets the resource of Kubernetes in etcd",
		Example: getExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			etcdclient, err := clientFromCmd(f)
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

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "yaml", "output format. One of: (yaml, json).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().Int64Var(&flags.ChunkSize, "chunk-size", 500, "chunk size of the list pager")
	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix to prepend to the resource")

	cmd.Flags().BoolVarP(&flags.Watch, "watch", "w", false, "after listing/getting the requested object, watch for changes")
	cmd.Flags().BoolVar(&flags.WatchOnly, "watch-only", false, "watch for changes to the requested object(s), without listing/getting first")
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

	printer := NewPrinter(os.Stdout, flags.Output)
	if printer == nil {
		return fmt.Errorf("invalid output format: %q", flags.Output)
	}

	opOpts := []client.OpOption{
		client.WithName(targetName, targetNamespace),
		client.WithGroupResource(targetGr),
		client.WithChunkSize(flags.ChunkSize),
		client.WithResponse(printer.Print),
	}

	if flags.Watch {
		if !flags.WatchOnly {
			rev, err := etcdclient.Get(ctx, flags.Prefix,
				opOpts...,
			)
			if err != nil {
				return err
			}
			opOpts = append(opOpts, client.WithRevision(rev))
		}

		err := etcdclient.Watch(ctx, flags.Prefix,
			opOpts...,
		)
		if err != nil {
			return err
		}
	} else {
		_, err := etcdclient.Get(ctx, flags.Prefix,
			opOpts...,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
