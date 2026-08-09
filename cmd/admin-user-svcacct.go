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

var adminUserSvcAcctSubcommands = []*cli.Command{
	adminUserSvcAcctAddCmd,
	adminUserSvcAcctListCmd,
	adminUserSvcAcctRemoveCmd,
	adminUserSvcAcctInfoCmd,
	adminUserSvcAcctSetCmd,
	adminUserSvcAcctEnableCmd,
	adminUserSvcAcctDisableCmd,
}

var adminUserSvcAcctCmd = &cli.Command{
	Name:            "svcacct",
	Usage:           "manage service accounts",
	Action:          mainAdminUserSvcAcct,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Commands:        adminUserSvcAcctSubcommands,
	HideHelpCommand: true,
}

// mainAdminUserSvcAcct is the handle for "mc admin user svcacct" command.
func mainAdminUserSvcAcct(_ context.Context, cmd *cli.Command) error {
	commandNotFound(cmd, adminUserSvcAcctSubcommands)
	return nil
}
