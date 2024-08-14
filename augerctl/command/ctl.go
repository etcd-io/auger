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

// Package command is a simple command line client for directly accessing data objects stored in etcd by Kubernetes.
package command

import (
	"github.com/spf13/cobra"

	"go.etcd.io/etcd/client/pkg/v3/transport"
)

type flagpole struct {
	Endpoints []string

	InsecureSkipVerify bool
	InsecureDiscovery  bool
	TLS                transport.TLSInfo

	User     string
	Password string
}

// NewCtlCommand returns a new cobra.Command for use ctl
func NewCtlCommand() *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Use:   "augerctl",
		Short: "A simple command line client for directly accessing data objects stored in etcd by Kubernetes.",
	}
	cmd.PersistentFlags().StringSliceVar(&flags.Endpoints, "endpoints", []string{"127.0.0.1:2379"}, "gRPC endpoints of etcd cluster")

	cmd.PersistentFlags().BoolVar(&flags.InsecureDiscovery, "insecure-discovery", true, "accept insecure SRV records describing cluster endpoints")
	cmd.PersistentFlags().BoolVar(&flags.InsecureSkipVerify, "insecure-skip-tls-verify", false, "skip server certificate verification")
	cmd.PersistentFlags().StringVar(&flags.TLS.CertFile, "cert", "", "path to the etcd client TLS cert file")
	cmd.PersistentFlags().StringVar(&flags.TLS.KeyFile, "key", "", "path to the etcd client TLS key file")
	cmd.PersistentFlags().StringVar(&flags.TLS.TrustedCAFile, "cacert", "", "path to the etcd client TLS CA cert file")
	cmd.PersistentFlags().StringVar(&flags.User, "user", "", "username for authentication, provide username[:password]")
	cmd.PersistentFlags().StringVar(&flags.Password, "password", "", "password for authentication, only available if --user has no password")

	cmd.AddCommand(
		newCtlGetCommand(flags),
	)
	return cmd
}
