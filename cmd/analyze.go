/*
Copyright 2022 The Kubernetes Authors.

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

package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/etcd-io/auger/pkg/data"
	"github.com/etcd-io/auger/pkg/scheme"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze kubernetes data from the boltdb '.db' files etcd persists to.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return analyze(os.Stdout, analyzeOpts.filename)
	},
}

type analyzeOptions struct {
	filename string
}

var analyzeOpts = &analyzeOptions{}

func init() {
	RootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().StringVarP(&analyzeOpts.filename, "file", "f", "", "Bolt DB '.db' filename")
}

func analyzeValidateAndRun() error {
	return analyze(os.Stdout, analyzeOpts.filename)
}

func analyze(out io.Writer, filename string) error {
	summaries, err := data.ListKeySummaries(scheme.Codecs, filename, []data.Filter{}, &data.KeySummaryProjection{HasKey: true, HasValue: false}, 0)
	if err != nil {
		return err
	}

	objectCounts := map[string]uint{}
	for _, s := range summaries {
		if s.TypeMeta != nil {
			objectCounts[fmt.Sprintf("%s/%s", s.TypeMeta.APIVersion, s.TypeMeta.Kind)]++
		}
	}

	type entry struct {
		count uint
		gvk   string
	}

	entries := []entry{}
	for k, v := range objectCounts {
		entries = append(entries, entry{gvk: k, count: v})
	}

	sort.Slice(entries[:], func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	var totalKeySize int
	var totalValueSize int
	var totalAllVersionsKeySize int
	var totalAllVersionsValueSize int
	for _, summary := range summaries {
		totalKeySize += summary.Stats.KeySize
		totalValueSize += summary.Stats.ValueSize
		totalAllVersionsKeySize += summary.Stats.AllVersionsKeySize
		totalAllVersionsValueSize += summary.Stats.AllVersionsValueSize
	}

	fmt.Fprintf(out, "Total kubernetes objects: %d\n", len(summaries))
	fmt.Fprintf(out, "Total (all revisions) storage used by kubernetes objects: %d\n", totalAllVersionsKeySize+totalAllVersionsValueSize)
	fmt.Fprintf(out, "Current (latest revision) storage used by kubernetes objects: %d\n", totalKeySize+totalValueSize)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Most common kubernetes types:")
	for i, entry := range entries {
		if i >= 10 {
			break
		}
		fmt.Fprintf(out, "\t%d\t%s\n", entry.count, entry.gvk)
	}

	sort.Slice(summaries[:], func(i, j int) bool {
		return summaries[i].Stats.AllVersionsValueSize > summaries[j].Stats.AllVersionsValueSize
	})
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Largest objects (byte size sum of all revisions):")
	printed := 0
	for _, summary := range summaries {
		if summary.TypeMeta == nil {
			continue
		}
		if printed >= 10 {
			break
		}
		fmt.Fprintf(out, "\t%d\t%s (%s/%s)\n", summary.Stats.AllVersionsValueSize, summary.Key, summary.TypeMeta.APIVersion, summary.TypeMeta.Kind)
		printed++
	}

	return nil
}
