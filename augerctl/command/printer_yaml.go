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
	"errors"
	"fmt"
	"io"

	"github.com/etcd-io/auger/pkg/client"
	"github.com/etcd-io/auger/pkg/encoding"
	"github.com/etcd-io/auger/pkg/scheme"
)

type yamlPrinter struct {
	w io.Writer
}

func (p *yamlPrinter) Print(kv *client.KeyValue) error {
	value := kv.Value
	if value == nil {
		value = kv.PrevValue
	}
	inMediaType, _, err := encoding.DetectAndExtract(value)
	if err != nil {
		_, err0 := fmt.Fprintf(p.w, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
		if err0 != nil {
			return errors.Join(err, err0)
		}
		return nil
	}
	data, _, err := encoding.Convert(scheme.Codecs, inMediaType, encoding.YamlMediaType, value)
	if err != nil {
		_, err0 := fmt.Fprintf(p.w, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
		if err0 != nil {
			return errors.Join(err, err0)
		}
		return nil
	}
	_, err = fmt.Fprintf(p.w, "---\n# %s | %s\n%s\n", kv.Key, inMediaType, data)
	if err != nil {
		return err
	}
	return nil
}
