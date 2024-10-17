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

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information
var (
	// AppVersion is the version of the application
	appVersion = "v0.0.0-dev"

	// BuildTime is the time the application was built
	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format

	// GitCommit is the commit hash of the build
	gitCommit string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print current version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Version:\t%s\n", appVersion)
		fmt.Printf("BuildTime:\t%s\n", buildDate)
		fmt.Printf("GitCommit:\t%s\n", gitCommit)
		fmt.Printf("GoVersion:\t%s\n", runtime.Version())
		fmt.Printf("Compiler:\t%s\n", runtime.Compiler)
		fmt.Printf("OS/Arch:\t%s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
