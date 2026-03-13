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
	"io"

	"github.com/etcd-io/auger/pkg/client"
	"github.com/etcd-io/auger/pkg/encoding"
	"github.com/etcd-io/auger/pkg/scheme"
)

type jsonPrinter struct {
	w io.Writer
}

func (p *jsonPrinter) Print(kv *client.KeyValue) error {
	value := kv.Value
	if value == nil {
		value = kv.PrevValue
	}
	inMediaType, _, err := encoding.DetectAndExtract(value)
	if err != nil {
		return err
	}
	data, _, err := encoding.Convert(scheme.Codecs, inMediaType, encoding.JsonMediaType, value)
	if err != nil {
		return err
	}
	_, err = p.w.Write(data)
	if err != nil {
		return err
	}
	return nil
}
