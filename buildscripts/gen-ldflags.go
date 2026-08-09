//go:build ignore
// +build ignore

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func genLDFlags(version, releaseTag string, date time.Time) string {
	copyrightYear := fmt.Sprintf("%d", date.Year())

	var ldflagsStr string
	ldflagsStr = "-s -w -X github.com/delta592/mc/cmd.Version=" + version + " "
	ldflagsStr = ldflagsStr + "-X github.com/delta592/mc/cmd.CopyrightYear=" + copyrightYear + " "
	ldflagsStr = ldflagsStr + "-X github.com/delta592/mc/cmd.ReleaseTag=" + releaseTag + " "
	ldflagsStr = ldflagsStr + "-X github.com/delta592/mc/cmd.CommitID=" + commitID() + " "
	ldflagsStr = ldflagsStr + "-X github.com/delta592/mc/cmd.ShortCommitID=" + commitID()[:12]
	return ldflagsStr
}

// describeVersion returns semver-style version strings from git describe output,
// e.g. v2.0.3+14 for 14 commits after tag v2.0.3, or v2.0.3 when exactly on a tag.
func describeVersion() (version, releaseTag string) {
	desc := strings.TrimSpace(gitOutput("describe", "--tags", "--always", "--dirty"))
	dirty := ""
	if strings.HasSuffix(desc, "-dirty") {
		dirty = ".dirty"
		desc = strings.TrimSuffix(desc, "-dirty")
	}

	if !strings.HasPrefix(desc, "v") {
		short := desc
		if strings.HasPrefix(short, "g") {
			short = short[1:]
		}
		releaseTag = "v0.0.0+" + short + dirty
		version = "0.0.0+" + short + dirty
		return version, releaseTag
	}

	parts := strings.Split(desc, "-")
	tag := parts[0]
	if len(parts) == 1 {
		releaseTag = tag + dirty
		version = strings.TrimPrefix(tag, "v") + dirty
		return version, releaseTag
	}

	commits := parts[1]
	releaseTag = tag + "+" + commits + dirty
	version = strings.TrimPrefix(tag, "v") + "+" + commits + dirty
	return version, releaseTag
}

// releaseTagFromTimestamp builds legacy RELEASE/DEVELOPMENT.<timestamp> tags.
func releaseTagFromTimestamp(version string) (string, time.Time) {
	relPrefix := "DEVELOPMENT"
	if prefix := os.Getenv("MC_RELEASE"); prefix != "" {
		relPrefix = prefix
	}

	relSuffix := ""
	if hotfix := os.Getenv("MC_HOTFIX"); hotfix != "" {
		relSuffix = hotfix
	}

	relTag := strings.ReplaceAll(version, " ", "-")
	relTag = strings.ReplaceAll(relTag, ":", "-")
	t, err := time.Parse("2006-01-02T15-04-05Z", relTag)
	if err != nil {
		panic(err)
	}

	relTag = strings.ReplaceAll(relTag, ",", "")
	relTag = relPrefix + "." + relTag
	if relSuffix != "" {
		relTag += "." + relSuffix
	}

	return relTag, t
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error running git ", args, ": ", err)
		os.Exit(1)
	}
	return string(out)
}

// commitID returns the abbreviated commit-id hash of the last commit.
func commitID() string {
	return strings.TrimSpace(gitOutput("log", "--format=%H", "-n1"))
}

func commitTime() time.Time {
	commitUnix := gitOutput("log", "--format=%cI", "-n1")

	t, err := time.Parse(time.RFC3339, strings.TrimSpace(commitUnix))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating git commit-time: ", err)
		os.Exit(1)
	}

	return t.UTC()
}

func main() {
	if os.Getenv("MC_RELEASE") != "" {
		version := commitTime().Format(time.RFC3339)
		if len(os.Args) > 1 {
			version = os.Args[1]
		}
		releaseTag, date := releaseTagFromTimestamp(version)
		fmt.Println(genLDFlags(version, releaseTag, date))
		return
	}

	if len(os.Args) > 1 {
		releaseTag, date := releaseTagFromTimestamp(os.Args[1])
		fmt.Println(genLDFlags(os.Args[1], releaseTag, date))
		return
	}

	version, releaseTag := describeVersion()
	fmt.Println(genLDFlags(version, releaseTag, commitTime()))
}
