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
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/delta592/mc/pkg/probe"
	"github.com/minio/madmin-go/v4"
	"github.com/urfave/cli/v3"
)

var supportTopDriveFlags = []cli.Flag{
	&cli.IntFlag{
		Name:    "count",
		Aliases: []string{"c"},
		Usage:   "show up to N drives",
		Value:   10,
	},
}

var supportTopDriveCmd = &cli.Command{
	Name:            "drive",
	Aliases:         []string{"disk"},
	Usage:           "show real-time drive metrics",
	Action:          mainSupportTopDrive,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportTopDriveFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display drive metrics
      {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkSupportTopDriveSyntax - validate all the passed arguments
func checkSupportTopDriveSyntax(cmd *cli.Command) {
	if cmd.Args().Len() == 0 || cmd.Args().Len() > 1 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

func mainSupportTopDrive(_ context.Context, cmd *cli.Command) error {
	checkSupportTopDriveSyntax(cmd)

	aliasedURL := cmd.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	info, e := client.ServerInfo(ctxt)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to initialize admin client.")

	var disks []madmin.Disk
	for _, srv := range info.Servers {
		disks = append(disks, srv.Disks...)
	}

	// MetricsOptions are options provided to Metrics call.
	opts := madmin.MetricsOptions{
		Type:     madmin.MetricsDisk,
		Interval: time.Second,
		ByDisk:   true,
		N:        cmd.Int("count"),
	}

	p := tea.NewProgram(initTopDriveUI(disks, cmd.Int("count")))
	go func() {
		out := func(m madmin.RealtimeMetrics) {
			for name, metric := range m.ByDisk {
				p.Send(topDriveResult{
					diskName: name,
					stats:    metric.IOStatsMinute,
				})
			}
		}

		e := client.Metrics(ctxt, opts, out)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to fetch top drives events")
		}
		p.Quit()
	}()

	if _, e := p.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch top drive events")
	}

	return nil
}
