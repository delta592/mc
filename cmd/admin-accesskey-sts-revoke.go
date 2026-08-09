// Copyright (c) 2015-2025 MinIO, Inc.
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

	json "github.com/delta592/mc/pkg/colorjson"
	"github.com/delta592/mc/pkg/probe"
	"github.com/minio/madmin-go/v4"
	"github.com/urfave/cli/v3"
)

var adminAccesskeySTSRevokeFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "all",
		Usage: "revoke all STS accounts for the specified user",
	},
	&cli.BoolFlag{
		Name:  "self",
		Usage: "revoke all STS accounts for the authenticated user",
	},
	&cli.StringFlag{
		Name:  "token-type",
		Usage: "specify the token type to revoke",
	},
}

var adminAccesskeySTSRevokeCmd = &cli.Command{
	Name:         "sts-revoke",
	Usage:        "revokes all STS accounts or specified types for the specified user",
	Action:       mainAdminAccesskeySTSRevoke,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminAccesskeySTSRevokeFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS USER [--all | --token-type TOKEN_TYPE]

  Exactly one of --all or --token-type must be specified.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Revoke all STS accounts for user "user1"
	 {{.Prompt}} {{.HelpName}} myminio user1 --all

  2. Revoke STS accounts of a token type "app-1" for user "user1"
	 {{.Prompt}} {{.HelpName}} myminio user1 --token-type app-1

  3. Revoke all STS accounts for the authenticated user
	 {{.Prompt}} {{.HelpName}} myminio --self

  4. Revoke STS accounts of a token type "app-1" for the authenticated user
	 {{.Prompt}} {{.HelpName}} myminio --self --token-type app-1
`,
}

type stsRevokeMessage struct {
	Status          string `json:"status"`
	User            string `json:"user"`
	TokenRevokeType string `json:"tokenRevokeType,omitempty"`
}

func (m stsRevokeMessage) String() string {
	userString := "user " + m.User
	if m.User == "" {
		userString = "authenticated user"
	}
	if m.TokenRevokeType == "" {
		return "Successfully revoked all STS accounts for " + userString
	}
	return "Successfully revoked all STS accounts of type " + m.TokenRevokeType + " for " + userString
}

func (m stsRevokeMessage) JSON() string {
	if m.Status == "" {
		m.Status = "success"
	}
	jsonMessageBytes, e := json.MarshalIndent(m, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// checkSTSRevokeSyntax - validate all the passed arguments
func checkSTSRevokeSyntax(cmd *cli.Command) {
	if cmd.Args().Len() > 2 || cmd.Args().Len() == 0 {
		showCommandHelpAndExit(cmd, 1)
	}

	if !cmd.Bool("self") && cmd.Args().Get(1) == "" {
		fatalIf(errInvalidArgument().Trace(), "Must specify user or use --self flag.")
	}

	if cmd.Bool("self") && cmd.Args().Get(1) != "" {
		fatalIf(errInvalidArgument().Trace(), "Cannot specify user with --self flag.")
	}

	if (!cmd.Bool("all") && cmd.String("token-type") == "") || (cmd.Bool("all") && cmd.String("token-type") != "") {
		fatalIf(errDummy().Trace(), "Exactly one of --all or --token-type must be specified.")
	}
}

// mainAdminAccesskeySTSRevoke is the handle for "mc admin accesskey sts-revoke" command.
func mainAdminAccesskeySTSRevoke(_ context.Context, cmd *cli.Command) error {
	checkSTSRevokeSyntax(cmd)

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)
	user := args.Get(1) // will be empty if --self flag is set
	tokenRevokeType := cmd.String("token-type")
	fullRevoke := cmd.Bool("all")

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	e := client.RevokeTokens(globalContext, madmin.RevokeTokensReq{
		User:            user,
		TokenRevokeType: tokenRevokeType,
		FullRevoke:      fullRevoke,
	})
	fatalIf(probe.NewError(e).Trace(args.Slice()...), "Unable to revoke tokens for %s", user)

	printMsg(stsRevokeMessage{
		User:            user,
		TokenRevokeType: tokenRevokeType,
	})

	return nil
}
