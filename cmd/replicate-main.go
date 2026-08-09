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

var replicateSubcommands = []*cli.Command{
	replicateAddCmd,
	replicateUpdateCmd,
	replicateListCmd,
	replicateStatusCmd,
	replicateResyncCmd,
	replicateExportCmd,
	replicateImportCmd,
	replicateRemoveCmd,
	replicateBacklogCmd,
}

var replicateCmd = &cli.Command{
	Name:            "replicate",
	Usage:           "configure server side bucket replication",
	HideHelpCommand: true,
	Action:          mainReplicate,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	Commands:        replicateSubcommands,
}

// mainReplicate is the handle for "mc replicate" command.
func mainReplicate(_ context.Context, cmd *cli.Command) error {
	commandNotFound(cmd, replicateSubcommands)
	return nil
	// Sub-commands like "list", "clear", "add" have their own main.
}
