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

// Package ctl is A simple command line client for directly access data objects stored in etcd by Kubernetes.
package ctl

import (
	"time"

	"github.com/spf13/cobra"

	"go.etcd.io/etcd/client/pkg/v3/transport"
)

type flagpole struct {
	Insecure              bool
	InsecureSkipVerify    bool
	InsecureDiscovery     bool
	Endpoints             []string
	DialTimeout           time.Duration
	CommandTimeOut        time.Duration
	KeepAliveTime         time.Duration
	KeepAliveTimeout      time.Duration
	DNSClusterServiceName string

	TLS transport.TLSInfo

	User     string
	Password string
}

// NewCtlCommand returns a new cobra.Command for use ctl
func NewCtlCommand() *cobra.Command {
	flags := &flagpole{}
	const (
		defaultDialTimeout      = 2 * time.Second
		defaultCommandTimeOut   = 5 * time.Second
		defaultKeepAliveTime    = 2 * time.Second
		defaultKeepAliveTimeOut = 6 * time.Second
	)

	cmd := &cobra.Command{
		Use:   "augerctl",
		Short: "A simple command line client for directly access data objects stored in etcd by Kubernetes.",
	}
	cmd.PersistentFlags().StringSliceVar(&flags.Endpoints, "endpoints", []string{"127.0.0.1:2379"}, "gRPC endpoints")

	cmd.PersistentFlags().DurationVar(&flags.DialTimeout, "dial-timeout", defaultDialTimeout, "dial timeout for client connections")
	cmd.PersistentFlags().DurationVar(&flags.CommandTimeOut, "command-timeout", defaultCommandTimeOut, "timeout for short running command (excluding dial timeout)")
	cmd.PersistentFlags().DurationVar(&flags.KeepAliveTime, "keepalive-time", defaultKeepAliveTime, "keepalive time for client connections")
	cmd.PersistentFlags().DurationVar(&flags.KeepAliveTimeout, "keepalive-timeout", defaultKeepAliveTimeOut, "keepalive timeout for client connections")

	cmd.PersistentFlags().BoolVar(&flags.Insecure, "insecure-transport", true, "disable transport security for client connections")
	cmd.PersistentFlags().BoolVar(&flags.InsecureDiscovery, "insecure-discovery", true, "accept insecure SRV records describing cluster endpoints")
	cmd.PersistentFlags().BoolVar(&flags.InsecureSkipVerify, "insecure-skip-tls-verify", false, "skip server certificate verification (CAUTION: this option should be enabled only for testing purposes)")
	cmd.PersistentFlags().StringVar(&flags.TLS.CertFile, "cert", "", "identify secure client using this TLS certificate file")
	cmd.PersistentFlags().StringVar(&flags.TLS.KeyFile, "key", "", "identify secure client using this TLS key file")
	cmd.PersistentFlags().StringVar(&flags.TLS.TrustedCAFile, "cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	cmd.PersistentFlags().StringVar(&flags.User, "user", "", "username[:password] for authentication (prompt if password is not supplied)")
	cmd.PersistentFlags().StringVar(&flags.Password, "password", "", "password for authentication (if this option is used, --user option shouldn't include password)")
	cmd.PersistentFlags().StringVarP(&flags.TLS.ServerName, "discovery-srv", "d", "", "domain name to query for SRV records describing cluster endpoints")
	cmd.PersistentFlags().StringVarP(&flags.DNSClusterServiceName, "discovery-srv-name", "", "", "service name to query when using DNS discovery")

	cmd.AddCommand(
		newCtlGetCommand(),
		newCtlDelCommand(),
		newCtlPutCommand(),
	)
	return cmd
}
