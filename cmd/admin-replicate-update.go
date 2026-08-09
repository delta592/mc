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
	"net/url"
	"strings"

	json "github.com/delta592/mc/pkg/colorjson"
	"github.com/delta592/mc/pkg/probe"
	"github.com/fatih/color"
	"github.com/minio/madmin-go/v4"
	"github.com/minio/pkg/v3/console"
	"github.com/urfave/cli/v3"
)

var adminReplicateUpdateFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "deployment-id",
		Usage: "deployment id of the site, should be a unique value",
	},
	&cli.StringFlag{
		Name:  "endpoint",
		Usage: "endpoint for the site",
	},
	&cli.StringFlag{
		Name:  "mode",
		Usage: "change mode of replication for this target, valid values are ['sync', 'async'].",
		Value: "",
	},
	&cli.StringFlag{
		Name:   "sync",
		Usage:  "enable synchronous replication for this target, valid values are ['enable', 'disable'].",
		Value:  "disable",
		Hidden: true, // deprecated Jul 2023
	},
	&cli.StringFlag{
		Name:  "bucket-bandwidth",
		Usage: "Set default bandwidth limit for bucket in bytes per second (K,B,G,T for metric and Ki,Bi,Gi,Ti for IEC units)",
	},
	&cli.BoolFlag{
		Name:  "disable-ilm-expiry-replication",
		Usage: "disable ILM expiry rules replication",
	},
	&cli.BoolFlag{
		Name:  "enable-ilm-expiry-replication",
		Usage: "enable ILM expiry rules replication",
	},
}

var adminReplicateUpdateCmd = &cli.Command{
	Name:         "update",
	Aliases:      []string{"edit"},
	Usage:        "modify endpoint of site participating in site replication",
	Action:       mainAdminReplicateUpdate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminReplicateUpdateFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS --deployment-id [DEPLOYMENT-ID] --endpoint [NEW-ENDPOINT]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Edit a site endpoint participating in cluster-level replication:
     {{.Prompt}} {{.HelpName}} myminio --deployment-id c1758167-4426-454f-9aae-5c3dfdf6df64 --endpoint https://minio2:9000

  2. Set default bucket bandwidth limit for replication from myminio to the peer cluster with deployment-id c1758167-4426-454f-9aae-5c3dfdf6df64
     {{.Prompt}} {{.HelpName}} myminio --deployment-id c1758167-4426-454f-9aae-5c3dfdf6df64 --bucket-bandwidth "2G"

  3. Disable replication of ILM expiry in cluster-level replication:
     {{.Prompt}} {{.HelpName}} myminio --disable-ilm-expiry-replication

  4. Enable replication of ILM expiry in cluster-level replication:
     {{.Prompt}} {{.HelpName}} myminio --enable-ilm-expiry-replication
`,
}

type updateSuccessMessage madmin.ReplicateEditStatus

func (m updateSuccessMessage) JSON() string {
	bs, e := json.MarshalIndent(madmin.ReplicateEditStatus(m), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (m updateSuccessMessage) String() string {
	v := madmin.ReplicateEditStatus(m)
	messages := []string{v.Status}

	if v.ErrDetail != "" {
		messages = append(messages, v.ErrDetail)
	}
	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func checkAdminReplicateUpdateSyntax(cmd *cli.Command) {
	// Check argument count
	argsNr := cmd.Args().Len()
	if argsNr < 1 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
	if argsNr != 1 {
		fatalIf(errInvalidArgument().Trace(cmd.Args().Tail()...),
			"Invalid arguments specified for edit command.")
	}
}

func mainAdminReplicateUpdate(_ context.Context, cmd *cli.Command) error {
	checkAdminReplicateUpdateSyntax(cmd)
	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if !cmd.IsSet("deployment-id") && !cmd.IsSet("disable-ilm-expiry-replication") && !cmd.IsSet("enable-ilm-expiry-replication") {
		fatalIf(errInvalidArgument(), "--deployment-id is a required flag")
	}
	if !cmd.IsSet("endpoint") && !cmd.IsSet("mode") && !cmd.IsSet("sync") && !cmd.IsSet("bucket-bandwidth") && !cmd.IsSet("disable-ilm-expiry-replication") && !cmd.IsSet("enable-ilm-expiry-replication") {
		fatalIf(errInvalidArgument(), "--endpoint, --mode, --bucket-bandwidth, --disable-ilm-expiry-replication or --enable-ilm-expiry-replication is a required flag")
	}
	if cmd.IsSet("mode") && cmd.IsSet("sync") {
		fatalIf(errInvalidArgument(), "either --sync or --mode flag should be specified")
	}
	if cmd.IsSet("disable-ilm-expiry-replication") && cmd.IsSet("enable-ilm-expiry-replication") {
		fatalIf(errInvalidArgument(), "either --disable-ilm-expiry-replication or --enable-ilm-expiry-replication flag should be specified")
	}
	if (cmd.IsSet("disable-ilm-expiry-replication") || cmd.IsSet("enable-ilm-expiry-replication")) && cmd.IsSet("deployment-id") {
		fatalIf(errInvalidArgument(), "--deployment-id should not be set with --disable-ilm-expiry-replication or --enable-ilm-expiry-replication")
	}

	var syncState string
	if cmd.IsSet("sync") { // for backward compatibility - deprecated Jul 2023
		syncState = strings.ToLower(cmd.String("sync"))
		switch syncState {
		case "enable", "disable":
		default:
			fatalIf(errInvalidArgument().Trace(args.Slice()...), "--sync can be either [enable|disable]")
		}
	}

	if cmd.IsSet("mode") {
		mode := strings.ToLower(cmd.String("mode"))
		switch mode {
		case "sync":
			syncState = "enable"
		case "async":
			syncState = "disable"
		default:
			fatalIf(errInvalidArgument().Trace(args.Slice()...), "--mode can be either [sync|async]")
		}
	}

	var bwDefaults madmin.BucketBandwidth
	if cmd.IsSet("bucket-bandwidth") {
		bandwidthStr := cmd.String("bucket-bandwidth")
		bandwidth, e := getBandwidthInBytes(bandwidthStr)
		fatalIf(probe.NewError(e).Trace(bandwidthStr), "invalid bandwidth value")

		bwDefaults.Limit = bandwidth
		bwDefaults.IsSet = true
	}
	var ep string
	if cmd.IsSet("endpoint") {
		parsedURL := cmd.String("endpoint")
		u, e := url.Parse(parsedURL)
		if e != nil {
			fatalIf(errInvalidArgument().Trace(parsedURL), "Unsupported URL format %v", e)
		}
		ep = u.String()
	}
	var opts madmin.SREditOptions
	opts.DisableILMExpiryReplication = cmd.Bool("disable-ilm-expiry-replication")
	opts.EnableILMExpiryReplication = cmd.Bool("enable-ilm-expiry-replication")
	res, e := client.SiteReplicationEdit(globalContext, madmin.PeerInfo{
		DeploymentID:     cmd.String("deployment-id"),
		Endpoint:         ep,
		SyncState:        madmin.SyncStatus(syncState),
		DefaultBandwidth: bwDefaults,
	}, opts)
	fatalIf(probe.NewError(e).Trace(args.Slice()...), "Unable to edit cluster replication site endpoint")

	printMsg(updateSuccessMessage(res))

	return nil
}
