/*
Copyright 2019 The Kubernetes Authors.

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

package data

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/etcd-io/auger/pkg/scheme"

	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/mvccpb"
)

const (
	dbFile        = "testdata/boltdb/db"
	dbWithCrdFile = "testdata/boltdb/db-with-crd"
)

func TestListKeySummariesFilters(t *testing.T) {
	cases := []struct {
		file         string
		name         string
		filters      []Filter
		expectedKeys []string
	}{
		{
			file:         dbFile,
			name:         "nofilters",
			filters:      []Filter{},
			expectedKeys: []string{"/registry/jobs/default/pi", "/registry/namespaces/default", "/registry/pods/default/pi-dqtsw", "compact_rev_key"},
		},
		{
			file:         dbFile,
			name:         "prefixfilter",
			filters:      []Filter{NewPrefixFilter("/registry/jobs")},
			expectedKeys: []string{"/registry/jobs/default/pi"},
		},
		{
			file:         dbFile,
			name:         "namespacefilter",
			filters:      []Filter{mustBuildFilter(&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"})},
			expectedKeys: []string{"/registry/jobs/default/pi", "/registry/pods/default/pi-dqtsw"},
		},
		{
			file:         dbFile,
			name:         "allfilters",
			filters:      []Filter{NewPrefixFilter("/registry/jobs"), mustBuildFilter(&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"})},
			expectedKeys: []string{"/registry/jobs/default/pi"},
		},
		{
			file: dbWithCrdFile,
			name: "crd",
			filters: []Filter{
				NewPrefixFilter("/registry/apiextensions.k8s.io/customresourcedefinitions"),
				mustBuildFilter(&FieldConstraint{lhs: ".TypeMeta.APIVersion", op: Equals, rhs: "apiextensions.k8s.io/v1beta1"}),
				mustBuildFilter(&FieldConstraint{lhs: ".TypeMeta.Kind", op: Equals, rhs: "CustomResourceDefinition"}),
			},
			expectedKeys: []string{"/registry/apiextensions.k8s.io/customresourcedefinitions/crontabs.stable.example.com"},
		},
		{
			file: dbWithCrdFile,
			name: "cr",
			filters: []Filter{
				NewPrefixFilter("/registry/stable.example.com/crontabs"),
				mustBuildFilter(&FieldConstraint{lhs: ".TypeMeta.APIVersion", op: Equals, rhs: "stable.example.com/v1"}),
				mustBuildFilter(&FieldConstraint{lhs: ".TypeMeta.Kind", op: Equals, rhs: "CronTab"}),
			},
			expectedKeys: []string{"/registry/stable.example.com/crontabs/default/my-new-cron-object"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			missingKeys := map[string]struct{}{}
			for _, key := range tt.expectedKeys {
				missingKeys[key] = struct{}{}
			}
			unexpectedKeys := map[string]struct{}{}
			results, err := ListKeySummaries(scheme.Codecs, tt.file, tt.filters, ProjectEverything, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, result := range results {
				if _, ok := missingKeys[result.Key]; ok {
					delete(missingKeys, result.Key)
				} else {
					unexpectedKeys[result.Key] = struct{}{}
				}
			}
			if len(unexpectedKeys) != 0 {
				t.Errorf("got %d unexpected keys: %v, expected none", len(unexpectedKeys), unexpectedKeys)
			}
			if len(missingKeys) != 0 {
				t.Errorf("got %d missing keys: %v, expected none", len(missingKeys), missingKeys)
			}
		})
	}
}

func TestListKeySummariesUsesKeyVersion(t *testing.T) {
	versionInfo := collectKeyVersionInfo(t, dbWithCrdFile)
	results, err := ListKeySummaries(scheme.Codecs, dbWithCrdFile, nil, ProjectEverything, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatalf("expected results for %s", dbWithCrdFile)
	}

	var diffKey string
	for _, result := range results {
		info, ok := versionInfo[result.Key]
		if !ok {
			t.Fatalf("missing version info for key %q", result.Key)
		}
		if result.Version != info.version {
			t.Fatalf("key %q reported version %d, expected %d", result.Key, result.Version, info.version)
		}
		if info.updateCount > 1 && info.modRevision != info.version {
			diffKey = result.Key
		}
	}
	if diffKey == "" {
		t.Fatalf("fixture %s must include a key with multiple versions and differing mod revision for this regression test", dbWithCrdFile)
	}
}

func TestParseFilters(t *testing.T) {
	cases := []struct {
		name      string
		rawFilter string
		expected  []*FieldConstraint
	}{
		{
			name:      "namespace-equals",
			rawFilter: ".Value.metadata.namespace=default",
			expected:  []*FieldConstraint{{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"}},
		},
		{
			name:      "2-filters",
			rawFilter: ".Value.metadata.namespace=default,.Value.metadata.name=example",
			expected: []*FieldConstraint{
				{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"},
				{lhs: ".Value.metadata.name", op: Equals, rhs: "example"},
			},
		},
		{
			name:      "whitespace",
			rawFilter: " .Value.metadata.namespace=default\t, .Value.metadata.name=example\n",
			expected: []*FieldConstraint{
				{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"},
				{lhs: ".Value.metadata.name", op: Equals, rhs: "example"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := ParseFilters(tt.rawFilter)
			if err != nil {
				t.Fatal(err)
			}
			unexpected := map[FieldConstraint]struct{}{}
			missing := map[FieldConstraint]struct{}{}

			for _, expected := range tt.expected {
				missing[*expected] = struct{}{}
			}

			for _, filter := range filters {
				fc := filter.(*FieldFilter).FieldConstraint
				if _, ok := missing[*fc]; ok {
					delete(missing, *fc)
				} else {
					unexpected[*fc] = struct{}{}
				}
			}
			if len(unexpected) != 0 {
				t.Errorf("got %d unexpected filters: %#+v, expected none", len(unexpected), unexpected)
			}
			if len(missing) != 0 {
				t.Errorf("got %d missing filters: %#+v, expected none", len(missing), missing)
			}
		})
	}
}

func TestListKeySummariesMissingKeyBucket(t *testing.T) {
	file := createTestDB(t, func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(metaBucket)
		return err
	})

	_, err := ListKeySummaries(scheme.Codecs, file, nil, ProjectEverything, 0)
	if err == nil {
		t.Fatalf("expected error when key bucket is missing")
	}
	if !strings.Contains(err.Error(), `missing "key" bucket`) {
		t.Fatalf("expected missing key bucket error, got: %v", err)
	}
}

func TestHashByRevisionMissingMetaBucket(t *testing.T) {
	file := createTestDB(t, nil)

	_, err := HashByRevision(file, 0)
	if err == nil {
		t.Fatalf("expected error when meta bucket is missing")
	}
	if !strings.Contains(err.Error(), `missing "meta" bucket`) {
		t.Fatalf("expected missing meta bucket error, got: %v", err)
	}
}

func mustBuildFilter(fc *FieldConstraint) Filter {
	filter, err := fc.BuildFilter()
	if err != nil {
		panic(err)
	}
	return filter
}

type keyVersionInfo struct {
	version     int64
	modRevision int64
	updateCount int
}

func collectKeyVersionInfo(t *testing.T, file string) map[string]keyVersionInfo {
	t.Helper()

	db, err := boltOpen(file)
	if err != nil {
		t.Fatalf("failed to open db file %s: %v", file, err)
	}
	defer db.Close()

	info := make(map[string]keyVersionInfo)
	err = walk(db, func(r revKey, kv *mvccpb.KeyValue) (bool, error) {
		key := string(kv.Key)
		if r.tombstone {
			delete(info, key)
			return false, nil
		}
		entry := info[key]
		entry.updateCount++
		if kv.Version >= entry.version {
			entry.version = kv.Version
			entry.modRevision = kv.ModRevision
		}
		info[key] = entry
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed to collect version info: %v", err)
	}
	return info
}

func createTestDB(t *testing.T, initTx func(tx *bolt.Tx) error) string {
	t.Helper()

	file := filepath.Join(t.TempDir(), "db")
	db, err := bolt.Open(file, 0o600, nil)
	if err != nil {
		t.Fatalf("failed to create db file %s: %v", file, err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		if initTx == nil {
			return nil
		}
		return initTx(tx)
	})
	if err != nil {
		t.Fatalf("failed to initialize db file %s: %v", file, err)
	}
	return file
}
