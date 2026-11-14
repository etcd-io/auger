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
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/etcd-io/auger/pkg/scheme"
	"google.golang.org/protobuf/proto"

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

func TestListKeySummariesShortRevisionKey(t *testing.T) {
	file := createTestDB(t, func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(keyBucket)
		if err != nil {
			return err
		}
		return b.Put([]byte("short"), []byte("value"))
	})

	_, err := ListKeySummaries(scheme.Codecs, file, nil, ProjectEverything, 0)
	if err == nil {
		t.Fatalf("expected error when a revision key is too short")
	}
	if !strings.Contains(err.Error(), "revision key must be at least") {
		t.Fatalf("expected short revision key error, got: %v", err)
	}
}

func TestListKeySummariesTombstone(t *testing.T) {
	const cm = `{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"%s"}}`
	file := createTestDB(t, func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(keyBucket)
		if err != nil {
			return err
		}
		for _, e := range []struct {
			main      int64
			tombstone bool
			kv        *mvccpb.KeyValue
		}{
			{2, false, &mvccpb.KeyValue{Key: []byte("/registry/configmaps/default/deleted"), Value: []byte(fmt.Sprintf(cm, "deleted")), Version: 1}},
			{3, false, &mvccpb.KeyValue{Key: []byte("/registry/configmaps/default/recreated"), Value: []byte(fmt.Sprintf(cm, "old")), Version: 1}},
			{4, true, &mvccpb.KeyValue{Key: []byte("/registry/configmaps/default/deleted")}},
			{5, true, &mvccpb.KeyValue{Key: []byte("/registry/configmaps/default/recreated")}},
			{6, false, &mvccpb.KeyValue{Key: []byte("/registry/configmaps/default/recreated"), Value: []byte(fmt.Sprintf(cm, "new")), Version: 1}},
		} {
			v, err := proto.Marshal(e.kv)
			if err != nil {
				return err
			}
			if err := b.Put(testRevBytes(e.main, e.tombstone), v); err != nil {
				return err
			}
		}
		return nil
	})

	results, err := ListKeySummaries(scheme.Codecs, file, nil, ProjectEverything, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected only the recreated key, got %d results", len(results))
	}
	got := results[0]
	if got.Key != "/registry/configmaps/default/recreated" {
		t.Fatalf("unexpected key %q", got.Key)
	}
	if got.Version != 1 || got.Stats.VersionCount != 1 {
		t.Errorf("expected version 1 and version count 1, got %d and %d", got.Version, got.Stats.VersionCount)
	}
	if got.TypeMeta == nil || got.TypeMeta.Kind != "ConfigMap" {
		t.Errorf("expected ConfigMap type meta, got %v", got.TypeMeta)
	}
	if value := got.ValueJSON(); !strings.Contains(value, `"name":"new"`) {
		t.Errorf("expected value of the recreated object, got %s", value)
	}
}

func testRevBytes(main int64, tombstone bool) []byte {
	b := make([]byte, revBytesLen, markedRevBytesLen)
	binary.BigEndian.PutUint64(b[0:8], uint64(main))
	b[8] = '_'
	if tombstone {
		b = append(b, markTombstone)
	}
	return b
}

func TestHashByRevisionShortCompactRevision(t *testing.T) {
	file := createTestDB(t, func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(metaBucket)
		if err != nil {
			return err
		}
		return b.Put(finishedCompactKeyName, []byte("short"))
	})

	_, err := HashByRevision(file, 0)
	if err == nil {
		t.Fatalf("expected error when the compact revision value is too short")
	}
	if !strings.Contains(err.Error(), "revision key must be at least") {
		t.Fatalf("expected short revision key error, got: %v", err)
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

func TestListKeySummariesUsesLatestValue(t *testing.T) {
	const cm = `{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"app","namespace":"default"},"data":{"mode":"%s"}}`
	const key = "/registry/configmaps/default/app"
	file := createTestDB(t, func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(keyBucket)
		if err != nil {
			return err
		}
		for main, mode := range map[int64]string{2: "old", 3: "new"} {
			v, err := proto.Marshal(&mvccpb.KeyValue{Key: []byte(key), Value: []byte(fmt.Sprintf(cm, mode)), Version: main - 1})
			if err != nil {
				return err
			}
			if err := b.Put(testRevBytes(main, false), v); err != nil {
				return err
			}
		}
		return nil
	})

	tests := []struct {
		name         string
		filters      []Filter
		revision     int64
		expectedMode string
	}{
		{name: "no filter", expectedMode: "new"},
		{name: "filter on the latest value", filters: []Filter{mustBuildFilter(&FieldConstraint{lhs: ".Value.data.mode", op: Equals, rhs: "new"})}, expectedMode: "new"},
		{name: "filter on an older value", filters: []Filter{mustBuildFilter(&FieldConstraint{lhs: ".Value.data.mode", op: Equals, rhs: "old"})}},
		{name: "older revision", revision: 2, expectedMode: "old"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ListKeySummaries(scheme.Codecs, file, tt.filters, ProjectEverything, tt.revision)
			if err != nil {
				t.Fatal(err)
			}
			if tt.expectedMode == "" {
				if len(results) != 0 {
					t.Fatalf("expected no results, got %s", results[0].ValueJSON())
				}
				return
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			if value := results[0].ValueJSON(); !strings.Contains(value, `"mode":"`+tt.expectedMode+`"`) {
				t.Errorf("expected the value with mode %q, got %s", tt.expectedMode, value)
			}
			expectedVersion := int64(2)
			if tt.revision == 2 {
				expectedVersion = 1
			}
			if results[0].Version != expectedVersion || results[0].Stats.VersionCount != int(expectedVersion) {
				t.Errorf("expected version and version count %d, got %d and %d", expectedVersion, results[0].Version, results[0].Stats.VersionCount)
			}
		})
	}
}
