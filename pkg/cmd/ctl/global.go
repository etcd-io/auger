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

// Copy from https://github.com/etcd-io/etcd/blob/main/etcdctl/ctlv3/command/global.go
// and made some change

package ctl

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/etcd-io/auger/pkg/client"
	"github.com/spf13/cobra"

	"go.etcd.io/etcd/client/pkg/v3/srv"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type secureCfg struct {
	cert       string
	key        string
	cacert     string
	serverName string

	insecureTransport  bool
	insecureSkipVerify bool
}

type authCfg struct {
	username string
	password string
}

type discoveryCfg struct {
	domain      string
	insecure    bool
	serviceName string
}

type clientConfig struct {
	endpoints        []string
	dialTimeout      time.Duration
	keepAliveTime    time.Duration
	keepAliveTimeout time.Duration
	scfg             *secureCfg
	acfg             *authCfg
}

func clientConfigFromCmd(cmd *cobra.Command) (*clientConfig, error) {
	var err error
	cfg := &clientConfig{}
	cfg.endpoints, err = endpointsFromCmd(cmd)
	if err != nil {
		return nil, err
	}

	cfg.dialTimeout, err = cmd.Flags().GetDuration("dial-timeout")
	if err != nil {
		return nil, err
	}
	cfg.keepAliveTime, err = cmd.Flags().GetDuration("keepalive-time")
	if err != nil {
		return nil, err
	}
	cfg.keepAliveTimeout, err = cmd.Flags().GetDuration("keepalive-timeout")
	if err != nil {
		return nil, err
	}
	cfg.scfg, err = secureCfgFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	cfg.acfg, err = authCfgFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func clientFromCmd(cmd *cobra.Command) (client.Client, error) {
	cfg, err := clientConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	return cfg.client()
}

func (cc *clientConfig) client() (client.Client, error) {
	cfg, err := newClientCfg(cc.endpoints, cc.dialTimeout, cc.keepAliveTime, cc.keepAliveTimeout, cc.scfg, cc.acfg)
	if err != nil {
		return nil, err
	}

	return client.NewClient(*cfg)
}

func newClientCfg(endpoints []string, dialTimeout, keepAliveTime, keepAliveTimeout time.Duration, scfg *secureCfg, acfg *authCfg) (*clientv3.Config, error) {
	// set tls if any one tls option set
	var cfgtls *transport.TLSInfo
	tlsinfo := transport.TLSInfo{}
	if scfg.cert != "" {
		tlsinfo.CertFile = scfg.cert
		cfgtls = &tlsinfo
	}

	if scfg.key != "" {
		tlsinfo.KeyFile = scfg.key
		cfgtls = &tlsinfo
	}

	if scfg.cacert != "" {
		tlsinfo.TrustedCAFile = scfg.cacert
		cfgtls = &tlsinfo
	}

	if scfg.serverName != "" {
		tlsinfo.ServerName = scfg.serverName
		cfgtls = &tlsinfo
	}

	cfg := &clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          dialTimeout,
		DialKeepAliveTime:    keepAliveTime,
		DialKeepAliveTimeout: keepAliveTimeout,
	}

	if cfgtls != nil {
		clientTLS, err := cfgtls.ClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.TLS = clientTLS
	}

	// if key/cert is not given but user wants secure connection, we
	// should still setup an empty tls configuration for gRPC to setup
	// secure connection.
	if cfg.TLS == nil && !scfg.insecureTransport {
		cfg.TLS = &tls.Config{}
	}

	// If the user wants to skip TLS verification then we should set
	// the InsecureSkipVerify flag in tls configuration.
	if scfg.insecureSkipVerify && cfg.TLS != nil {
		cfg.TLS.InsecureSkipVerify = true
	}

	if acfg != nil {
		cfg.Username = acfg.username
		cfg.Password = acfg.password
	}

	return cfg, nil
}

func secureCfgFromCmd(cmd *cobra.Command) (*secureCfg, error) {
	cert, key, cacert, err := keyAndCertFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	insecureTr, err := cmd.Flags().GetBool("insecure-transport")
	if err != nil {
		return nil, err
	}
	skipVerify, err := cmd.Flags().GetBool("insecure-skip-tls-verify")
	if err != nil {
		return nil, err
	}
	discoveryCfg, err := discoveryCfgFromCmd(cmd)
	if err != nil {
		return nil, err
	}

	if discoveryCfg.insecure {
		discoveryCfg.domain = ""
	}

	return &secureCfg{
		cert:       cert,
		key:        key,
		cacert:     cacert,
		serverName: discoveryCfg.domain,

		insecureTransport:  insecureTr,
		insecureSkipVerify: skipVerify,
	}, nil
}

func keyAndCertFromCmd(cmd *cobra.Command) (cert, key, cacert string, err error) {
	if cert, err = cmd.Flags().GetString("cert"); err != nil {
		return "", "", "", err
	}
	if cert == "" && cmd.Flags().Changed("cert") {
		return "", "", "", errors.New("empty string is passed to --cert option")
	}

	if key, err = cmd.Flags().GetString("key"); err != nil {
		return "", "", "", err
	}
	if key == "" && cmd.Flags().Changed("key") {
		return "", "", "", errors.New("empty string is passed to --key option")
	}

	if cacert, err = cmd.Flags().GetString("cacert"); err != nil {
		return "", "", "", err
	}
	if cacert == "" && cmd.Flags().Changed("cacert") {
		return "", "", "", errors.New("empty string is passed to --cacert option")
	}

	return cert, key, cacert, nil
}

func authCfgFromCmd(cmd *cobra.Command) (*authCfg, error) {
	userFlag, err := cmd.Flags().GetString("user")
	if err != nil {
		return nil, err
	}
	passwordFlag, err := cmd.Flags().GetString("password")
	if err != nil {
		return nil, err
	}

	if userFlag == "" {
		return nil, nil
	}

	var cfg authCfg

	if passwordFlag == "" {
		splitted := strings.SplitN(userFlag, ":", 2)
		if len(splitted) < 2 {
			cfg.username = userFlag
			cfg.password, err = speakeasy.Ask("Password: ")
			if err != nil {
				return nil, err
			}
		} else {
			cfg.username = splitted[0]
			cfg.password = splitted[1]
		}
	} else {
		cfg.username = userFlag
		cfg.password = passwordFlag
	}

	return &cfg, nil
}

func discoveryCfgFromCmd(cmd *cobra.Command) (*discoveryCfg, error) {
	domain, err := cmd.Flags().GetString("discovery-srv")
	if err != nil {
		return nil, err
	}
	insecure, err := cmd.Flags().GetBool("insecure-discovery")
	if err != nil {
		return nil, err
	}
	serviceName, err := cmd.Flags().GetString("discovery-srv-name")
	if err != nil {
		return nil, err
	}
	return &discoveryCfg{
		domain:      domain,
		insecure:    insecure,
		serviceName: serviceName,
	}, nil
}

func endpointsFromCmd(cmd *cobra.Command) ([]string, error) {
	eps, err := endpointsFromFlagValue(cmd)
	if err != nil {
		return nil, err
	}
	// If domain discovery returns no endpoints, check endpoints flag
	if len(eps) == 0 {
		eps, err = cmd.Flags().GetStringSlice("endpoints")
		if err == nil {
			for i, ip := range eps {
				eps[i] = strings.TrimSpace(ip)
			}
		}
	}
	return eps, err
}

func endpointsFromFlagValue(cmd *cobra.Command) ([]string, error) {
	discoveryCfg, err := discoveryCfgFromCmd(cmd)
	if err != nil {
		return nil, err
	}

	// If we still don't have domain discovery, return nothing
	if discoveryCfg.domain == "" {
		return []string{}, nil
	}

	srvs, err := srv.GetClient("etcd-client", discoveryCfg.domain, discoveryCfg.serviceName)
	if err != nil {
		return nil, err
	}
	eps := srvs.Endpoints
	if discoveryCfg.insecure {
		return eps, err
	}
	// strip insecure connections
	ret := []string{}
	for _, ep := range eps {
		if strings.HasPrefix(ep, "http://") {
			fmt.Fprintf(os.Stderr, "ignoring discovered insecure endpoint %q\n", ep)
			continue
		}
		ret = append(ret, ep)
	}
	return ret, err
}
