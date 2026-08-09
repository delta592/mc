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

var quotaInfoCmd = &cli.Command{
	Name:         "info",
	Usage:        "show bucket quota",
	Action:       mainQuotaInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display bucket quota configured for "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkQuotaInfoSyntax - validate all the passed arguments
func checkQuotaInfoSyntax(cmd *cli.Command) {
	if cmd.Args().Len() == 0 || cmd.Args().Len() > 1 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

// mainQuotaInfo is the handler for "mc quota info" command.
func mainQuotaInfo(_ context.Context, cmd *cli.Command) error {
	checkQuotaInfoSyntax(cmd)

	console.SetColor("QuotaMessage", color.New(color.FgGreen))
	console.SetColor("QuotaInfo", color.New(color.FgCyan))

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	_, targetURL := url2Alias(args.Get(0))
	qCfg, e := client.GetBucketQuota(globalContext, targetURL)
	fatalIf(probe.NewError(e).Trace(args.Slice()...), "Unable to get bucket quota")
	sz := qCfg.Size
	printMsg(quotaMessage{
		op:        cmd.Name,
		Bucket:    targetURL,
		Quota:     sz,
		QuotaType: string(qCfg.Type),
		Status:    "success",
	})

	return nil
}
