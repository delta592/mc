// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"

	json "github.com/delta592/mc/pkg/colorjson"
	"github.com/delta592/mc/pkg/probe"
	"github.com/fatih/color"
	"github.com/minio/pkg/v3/console"
	"github.com/urfave/cli/v3"
)

var versionSuspendCmd = &cli.Command{
	Name:         "suspend",
	Usage:        "suspend bucket versioning",
	Action:       mainVersionSuspend,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS/BUCKET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Suspend versioning on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkVersionSuspendSyntax - validate all the passed arguments
func checkVersionSuspendSyntax(cmd *cli.Command) {
	if cmd.Args().Len() != 1 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

type versionSuspendMessage struct {
	Op         string
	Status     string `json:"status"`
	URL        string `json:"url"`
	Versioning struct {
		Status    string `json:"status"`
		MFADelete string `json:"MFADelete"`
	} `json:"versioning"`
}

func (v versionSuspendMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v versionSuspendMessage) String() string {
	return console.Colorize("versionSuspendMessage", fmt.Sprintf("%s versioning is suspended", v.URL))
}

func mainVersionSuspend(_ context.Context, cmd *cli.Command) error {
	ctx, cancelVersionSuspend := context.WithCancel(globalContext)
	defer cancelVersionSuspend()

	console.SetColor("versionSuspendMessage", color.New(color.FgGreen))

	checkVersionSuspendSyntax(cmd)

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	fatalIf(client.SetVersion(ctx, "suspend", nil, false), "Unable to suspend versioning")
	printMsg(versionSuspendMessage{
		Op:     cmd.Name,
		Status: "success",
		URL:    aliasedURL,
	})
	return nil
}
