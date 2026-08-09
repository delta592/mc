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
	"strings"

	"github.com/urfave/cli/v3"
)

var adminConsoleFlags = []cli.Flag{
	&cli.IntFlag{
		Name:    "limit",
		Aliases: []string{"l"},
		Usage:   "show last n log entries",
		Value:   10,
	},
	&cli.StringFlag{
		Name:    "type",
		Aliases: []string{"t"},
		Usage:   "list error logs by type. Valid options are '[minio, application, all]'",
		Value:   "all",
	},
}

var adminConsoleCmd = &cli.Command{
	Name:               "console",
	Usage:              "show MinIO logs",
	Action:             mainAdminConsole,
	OnUsageError:       onUsageError,
	Before:             setGlobalsFromContext,
	Flags:              append(adminConsoleFlags, globalFlags...),
	Hidden:             true,
	HideHelpCommand:    true,
	CustomHelpTemplate: "This command is not supported now and replaced by 'admin logs' command. Please use 'mc admin logs'.\n",
}

// mainAdminConsole - the entry function of console command
func mainAdminConsole(_ context.Context, cmd *cli.Command) error {
	newCmd := []string{"mc admin logs"}

	var flgStr string

	if cmd.IsSet("limit") {
		flgStr = fmt.Sprintf("%s %d", "--last", cmd.Int("limit"))
		newCmd = append(newCmd, flgStr)
	}

	if cmd.IsSet("type") {
		flgStr = fmt.Sprintf("%s %s", "--type", strings.ToLower(cmd.String("type")))
		newCmd = append(newCmd, flgStr)
	}
	newCmd = append(newCmd, cmd.Args().Slice()...)

	deprecatedError(strings.Join(newCmd, " "))
	return nil
}
