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

var adminPolicyEntitiesFlags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "user",
		Aliases: []string{"u"},
		Usage:   "list policies associated with user(s)",
	},
	&cli.StringSliceFlag{
		Name:    "group",
		Aliases: []string{"g"},
		Usage:   "list policies associated with group(s)",
	},
	&cli.StringSliceFlag{
		Name:    "policy",
		Aliases: []string{"p"},
		Usage:   "list users or groups associated with policy",
	},
}

var adminPolicyEntitiesCmd = &cli.Command{
	Name:         "entities",
	Usage:        "list policy association entities",
	Action:       mainAdminPolicyEntities,
	Before:       setGlobalsFromContext,
	Flags:        append(adminPolicyEntitiesFlags, globalFlags...),
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
  
USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List all entities associated with all policies
     {{.Prompt}} {{.HelpName}} play/
  2. List all entities associated with the policies 'finteam-policy' and 'mlteam-policy'
     {{.Prompt}} {{.HelpName}} play/ --policy finteam-policy --policy mlteam-policy
  3. List all policies associated with a pair of user entities
     {{.Prompt}} {{.HelpName}} play/ --user bob --user james
  4. List all policies associated with a pair of group entities
     {{.Prompt}} {{.HelpName}} play/ --group auditors --group accounting
  5. List all entities associated with a policy, group and user
     {{.Prompt}} {{.HelpName}} play/ \
              --policy finteam-policy --user bobfisher --group consulting
`,
}

// mainAdminPolicyEntities is the handler for "mc admin policy entities" command.
func mainAdminPolicyEntities(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		showCommandHelpAndExit(cmd, 1)
	}

	usersToQuery := cmd.StringSlice("user")
	groupsToQuery := cmd.StringSlice("group")
	policiesToQuery := cmd.StringSlice("policy")

	args := cmd.Args()

	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	res, e := client.GetPolicyEntities(globalContext,
		madmin.PolicyEntitiesQuery{
			Users:  usersToQuery,
			Groups: groupsToQuery,
			Policy: policiesToQuery,
		})
	fatalIf(probe.NewError(e), "Unable to fetch policy entities")

	printMsg(policyEntitiesFrom(res))
	return nil
}
