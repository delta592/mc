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

var adminConfigRestoreCmd = &cli.Command{
	Name:         "restore",
	Usage:        "rollback back changes to a specific config history",
	Before:       setGlobalsFromContext,
	Action:       mainAdminConfigRestore,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET RESTOREID

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Restore 'restore-id' history key value on MinIO server.
     {{.Prompt}} {{.HelpName}} play/ <restore-id>
`,
}

// configRestoreMessage container to hold locks information.
type configRestoreMessage struct {
	Status      string `json:"status"`
	RestoreID   string `json:"restoreID"`
	targetAlias string
}

// String colorized service status message.
func (u configRestoreMessage) String() (msg string) {
	suggestion := fmt.Sprintf("mc admin service restart %s", u.targetAlias)
	msg += console.Colorize("ConfigRestoreMessage",
		fmt.Sprintf("Please restart your server with `%s`.\n", suggestion))
	msg += console.Colorize("ConfigRestoreMessage", "Restored "+u.RestoreID+" kv successfully.")
	return msg
}

// JSON jsonified service status Message message.
func (u configRestoreMessage) JSON() string {
	u.Status = "success"
	statusJSONBytes, e := json.MarshalIndent(u, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// checkAdminConfigRestoreSyntax - validate all the passed arguments
func checkAdminConfigRestoreSyntax(cmd *cli.Command) {
	if !cmd.Args().Present() || cmd.Args().Len() > 2 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

func mainAdminConfigRestore(_ context.Context, cmd *cli.Command) error {
	checkAdminConfigRestoreSyntax(cmd)

	console.SetColor("ConfigRestoreMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call get config API
	fatalIf(probe.NewError(client.RestoreConfigHistoryKV(globalContext, args.Get(1))), "Unable to restore server configuration.")

	// Print
	printMsg(configRestoreMessage{
		RestoreID:   args.Get(1),
		targetAlias: aliasedURL,
	})

	return nil
}
