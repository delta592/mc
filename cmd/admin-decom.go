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

	"github.com/urfave/cli/v3"
)

var adminDecommissionSubcommands = []*cli.Command{
	adminDecommissionStartCmd,
	adminDecommissionStatusCmd,
	adminDecommissionCancelCmd,
}

var adminDecommissionCmd = &cli.Command{
	Name:            "decommission",
	Aliases:         []string{"decom"},
	Usage:           "manage MinIO server pool decommissioning",
	Action:          mainAdminDecommission,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Commands:        adminDecommissionSubcommands,
	HideHelpCommand: true,
}

// mainAdminDecommission is the handle for "mc admin decommission" command.
func mainAdminDecommission(_ context.Context, cmd *cli.Command) error {
	commandNotFound(cmd, adminDecommissionSubcommands)
	return nil
	// Sub-commands like "get", "set" have their own main.
}
