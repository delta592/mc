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
	"github.com/minio/madmin-go/v4"
	"github.com/urfave/cli/v3"
)

var adminTierRmFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:   "force",
		Usage:  "forcefully remove the specified tier",
		Hidden: true,
	},
	&cli.BoolFlag{
		Name:   "dangerous",
		Usage:  "additional flag to be required in addition to force flag",
		Hidden: true,
	},
}

var adminTierRmCmd = &cli.Command{
	Name:         "remove",
	Aliases:      []string{"rm"},
	Usage:        "remove an empty remote tier",
	Action:       mainAdminTierRm,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminTierRmFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS NAME

NAME:
  Name of remote tier target. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove an empty tier by name 'WARM-TIER':
     {{.Prompt}} {{.HelpName}} myminio WARM-TIER
`,
}

func mainAdminTierRm(_ context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	nArgs := args.Len()
	if nArgs < 2 {
		showCommandHelpAndExit(cmd, 1)
	}
	if nArgs != 2 {
		fatalIf(errInvalidArgument().Trace(args.Tail()...),
			"Incorrect number of arguments for tier remove command.")
	}

	aliasedURL := args.Get(0)
	tierName := args.Get(1)
	if tierName == "" {
		fatalIf(errInvalidArgument(), "Tier name can't be empty")
	}

	if cmd.Bool("force") && !cmd.Bool("dangerous") {
		fatalIf(errInvalidArgument(), "This operation results in an irreversible disconnection from the specified remote tier. If you are really sure, retry this command with ‘--force’ and ‘--dangerous’ flags.")
	}

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	e := client.RemoveTierV2(globalContext, tierName, madmin.RemoveTierOpts{Force: cmd.Bool("force")})
	fatalIf(probe.NewError(e).Trace(args.Slice()...), "Unable to remove remote tier target")

	printMsg(&tierMessage{
		op:       cmd.Name,
		Status:   "success",
		TierName: tierName,
	})
	return nil
}
