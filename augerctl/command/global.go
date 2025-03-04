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
	"crypto/tls"
	"errors"
	"strings"

	"github.com/etcd-io/auger/pkg/client"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func clientFromCmd(f *flagpole) (client.Client, error) {
	cfg, err := clientConfigFromCmd(f)
	if err != nil {
		return nil, err
	}

	cli, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	return client.NewClient(cli), nil
}

func clientConfigFromCmd(f *flagpole) (clientv3.Config, error) {
	cfg := clientv3.Config{
		Endpoints: f.Endpoints,
	}

	if !f.TLS.Empty() {
		clientTLS, err := f.TLS.ClientConfig()
		if err != nil {
			return clientv3.Config{}, err
		}
		cfg.TLS = clientTLS
	}

	// if key/cert is not given but user wants secure connection, we
	// should still setup an empty tls configuration for gRPC to setup
	// secure connection.
	if cfg.TLS == nil && !f.InsecureDiscovery {
		cfg.TLS = &tls.Config{}
	}

	// If the user wants to skip TLS verification then we should set
	// the InsecureSkipVerify flag in tls configuration.
	if cfg.TLS != nil && f.InsecureSkipVerify {
		cfg.TLS.InsecureSkipVerify = true
	}

	if f.User != "" {
		if f.Password == "" {
			splitted := strings.SplitN(f.User, ":", 2)
			if len(splitted) < 2 {
				return clientv3.Config{}, errors.New("password is missing")
			}
			cfg.Username = splitted[0]
			cfg.Password = splitted[1]
		} else {
			cfg.Username = f.User
			cfg.Password = f.Password
		}
	}

	return cfg, nil
}
