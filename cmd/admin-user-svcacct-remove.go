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

var adminUserSvcAcctRemoveCmd = &cli.Command{
	Name:         "remove",
	Aliases:      []string{"rm"},
	Usage:        "remove a service account",
	Action:       mainAdminUserSvcAcctRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS SERVICE-ACCOUNT

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove a service account 'J123C4ZXEQN8RK6ND35I' from MinIO server.
     {{.Prompt}} {{.HelpName}} myminio/ J123C4ZXEQN8RK6ND35I
`,
}

// checkAdminUserSvcAcctRemoveSyntax - validate all the passed arguments
func checkAdminUserSvcAcctRemoveSyntax(cmd *cli.Command) {
	if cmd.Args().Len() != 2 {
		showCommandHelpAndExit(cmd, 1)
	}
}

// mainAdminUserSvcAcctRemove is the handle for "mc admin user svcacct rm" command.
func mainAdminUserSvcAcctRemove(_ context.Context, cmd *cli.Command) error {
	console.SetColor("AccMessage", color.New(color.FgGreen))

	checkAdminUserSvcAcctRemoveSyntax(cmd)

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)
	svcAccount := args.Get(1)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.DeleteServiceAccount(globalContext, svcAccount)
	fatalIf(probe.NewError(e).Trace(args.Slice()...), "Unable to remove the specified service account")

	printMsg(acctMessage{
		op:        svcAccOpRemove,
		AccessKey: svcAccount,
	})

	return nil
}
