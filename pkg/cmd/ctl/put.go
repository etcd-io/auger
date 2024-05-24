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
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/etcd-io/auger/pkg/client"
	"github.com/etcd-io/auger/pkg/encoding"
	"github.com/etcd-io/auger/pkg/scheme"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type putFlagpole struct {
	Namespace string
	Output    string
	Path      string
	Prefix    string
}

func newCtlPutCommand() *cobra.Command {
	flags := &putFlagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "put [resource] [name]",
		Short: "Puts the resource of k8s in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			etcdclient, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			err = putCommand(cmd.Context(), etcdclient, flags, args)

			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "key", "output format. One of: (key, none).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix to prepend to the resource")
	cmd.Flags().StringVar(&flags.Path, "path", "", "path of the file")
	return cmd
}

func putCommand(ctx context.Context, etcdclient client.Client, flags *putFlagpole, args []string) error {
	var reader io.Reader
	var err error
	switch flags.Path {
	case "-":
		reader = os.Stdin
	case "":
		return fmt.Errorf("path is required")
	default:
		reader, err = os.Open(flags.Path)
		if err != nil {
			return err
		}
	}

	var wantGr *schema.GroupResource
	var wantName string
	if len(args) != 0 {
		// TODO: Support get information from CRD and scheme.Codecs
		//       Support short name
		//       Check for namespaced

		gr := schema.ParseGroupResource(args[0])
		if gr.Empty() {
			return fmt.Errorf("invalid resource %q", args[0])
		}
		wantGr = &gr
		if len(args) >= 2 {
			wantName = args[1]
		}
	}

	start := time.Now()

	var count int
	var response func(kv *client.KeyValue) error
	if flags.Output == "key" {
		//nolint:unparam
		response = func(kv *client.KeyValue) error {
			count++
			if kv != nil {
				fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			}
			return nil
		}
	}

	err = decodeToUnstructured(reader, func(obj *unstructured.Unstructured) error {
		targetName := obj.GetName()
		if targetName == "" {
			// There will be some unnamed hidden resources, which we should also ignore.
			return nil
		}

		// TODO: Use a safe way to convert GVK to GVR
		//       Verify that all built-in resources conform to this rule
		//       For custom resources try to get information from the CRD
		targetGvr, _ := meta.UnsafeGuessKindToResource(obj.GroupVersionKind())

		targetGr := targetGvr.GroupResource()
		targetNamespace := obj.GetNamespace()

		if targetNamespace != "" && flags.Namespace != "" && targetNamespace != flags.Namespace {
			return nil
		}

		if wantGr != nil && *wantGr != targetGr {
			return nil
		}

		if wantName != "" && wantName != targetName {
			return nil
		}

		if targetName == "" {
			return nil
		}

		mediaType, err := client.MediaTypeFromGR(targetGr)
		if err != nil {
			return err
		}

		t := obj.GetCreationTimestamp()
		if t.IsZero() {
			obj.SetCreationTimestamp(metav1.Time{Time: start})
		}

		obj.SetResourceVersion("")
		obj.SetSelfLink("")

		data, err := obj.MarshalJSON()
		if err != nil {
			return err
		}

		data, _, err = encoding.Convert(scheme.Codecs, encoding.JsonMediaType, mediaType, data)
		if err != nil {
			return err
		}

		opOpts := []client.OpOption{
			client.WithName(targetName, targetNamespace),
			client.WithGR(targetGr),
		}

		if response != nil {
			opOpts = append(opOpts,
				client.WithResponse(response),
				client.WithKeysOnly(),
			)
		}

		err = etcdclient.Put(ctx, flags.Prefix, data,
			opOpts...,
		)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if flags.Output == "key" {
		fmt.Fprintf(os.Stderr, "put %d keys\n", count)
	}
	return nil
}

func decodeToUnstructured(reader io.Reader, visitFunc func(obj *unstructured.Unstructured) error) error {
	d := yaml.NewYAMLToJSONDecoder(reader)

	for {
		obj := &unstructured.Unstructured{}
		err := d.Decode(&obj)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		if obj.IsList() {
			err = obj.EachListItem(func(object runtime.Object) error {
				obj := object.(*unstructured.Unstructured)
				if len(obj.Object) == 0 {
					return nil
				}
				return visitFunc(object.(*unstructured.Unstructured))
			})
			if err != nil {
				return err
			}
		} else {
			if len(obj.Object) == 0 {
				continue
			}
			err = visitFunc(obj)
			if err != nil {
				return err
			}
		}
	}
}
