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

	"github.com/delta592/mc/pkg/probe"
	"github.com/fatih/color"
	"github.com/minio/pkg/v3/console"
	"github.com/urfave/cli/v3"
)

var adminPolicyRemoveCmd = &cli.Command{
	Name:         "remove",
	Aliases:      []string{"rm"},
	Usage:        "remove an IAM policy",
	Action:       mainAdminPolicyRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET POLICYNAME

POLICYNAME:
  Name of the canned policy on MinIO server.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove 'writeonly' policy on MinIO server.
     {{.Prompt}} {{.HelpName}} myminio writeonly
`,
}

// checkAdminPolicyRemoveSyntax - validate all the passed arguments
func checkAdminPolicyRemoveSyntax(cmd *cli.Command) {
	if cmd.Args().Len() != 2 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

// mainAdminPolicyRemove is the handle for "mc admin policy remove" command.
func mainAdminPolicyRemove(_ context.Context, cmd *cli.Command) error {
	checkAdminPolicyRemoveSyntax(cmd)

	console.SetColor("PolicyMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	fatalIf(probe.NewError(client.RemoveCannedPolicy(globalContext, args.Get(1))).Trace(args.Slice()...), "Unable to remove policy")

	printMsg(userPolicyMessage{
		op:     cmd.Name,
		Policy: args.Get(1),
	})

	return nil
}
