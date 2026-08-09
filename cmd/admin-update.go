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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	json "github.com/delta592/mc/pkg/colorjson"
	"github.com/delta592/mc/pkg/probe"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/minio/madmin-go/v4"
	"github.com/minio/pkg/v3/console"
	"github.com/urfave/cli/v3"
)

var adminUpdateFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "yes",
		Aliases: []string{"y"},
		Usage:   "Confirms the server update",
	},
}

var adminServerUpdateCmd = &cli.Command{
	Name:         "update",
	Usage:        "update all MinIO servers",
	Action:       mainAdminServerUpdate,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminUpdateFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update MinIO server represented by its alias 'play'.
     {{.Prompt}} {{.HelpName}} play/

  2. Update all MinIO servers in a distributed setup, represented by its alias 'mydist'.
     {{.Prompt}} {{.HelpName}} mydist/
`,
}

// serverUpdateMessage is container for ServerUpdate success and failure messages.
type serverUpdateMessage struct {
	Status             string                    `json:"status"`
	ServerURL          string                    `json:"serverURL"`
	ServerUpdateStatus madmin.ServerUpdateStatus `json:"serverUpdateStatus"`
}

// String colorized serverUpdate message.
func (s serverUpdateMessage) String() string {
	var rows []table.Row
	for _, peerRes := range s.ServerUpdateStatus.Results {
		errStr := fmt.Sprintf("upgraded server from %s to %s: %s", peerRes.CurrentVersion, peerRes.UpdatedVersion, tickCell)
		if peerRes.Err != "" {
			errStr = peerRes.Err
		} else if len(peerRes.WaitingDrives) > 0 {
			errStr = fmt.Sprintf("%d drives are hung, process was upgraded. However OS reboot is recommended.", len(peerRes.WaitingDrives))
		}
		rows = append(rows, table.Row{peerRes.Host, errStr})
	}

	t := table.NewWriter()
	var s1 strings.Builder
	s1.WriteString("Server update request sent successfully `" + s.ServerURL + "`\n")

	t.SetOutputMirror(&s1)
	t.SetColumnConfigs([]table.ColumnConfig{{Align: text.AlignCenter}})

	t.AppendHeader(table.Row{"Host", "Status"})
	t.AppendRows(rows)
	t.SetStyle(table.StyleLight)
	t.Render()

	return console.Colorize("ServiceRestart", s1.String())
}

// JSON jsonified server update message.
func (s serverUpdateMessage) JSON() string {
	serverUpdateJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serverUpdateJSONBytes)
}

// checkAdminServerUpdateSyntax - validate all the passed arguments
func checkAdminServerUpdateSyntax(cmd *cli.Command) {
	if cmd.Args().Len() == 0 || cmd.Args().Len() > 2 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

func mainAdminServerUpdate(_ context.Context, cmd *cli.Command) error {
	// Validate serivce update syntax.
	checkAdminServerUpdateSyntax(cmd)

	// Set color.
	console.SetColor("ServerUpdate", color.New(color.FgGreen, color.Bold))

	// Get the alias parameter from cli
	args := cmd.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	updateURL := args.Get(1)

	autoConfirm := cmd.Bool("yes")

	if isTerminal() && !autoConfirm {
		fmt.Printf("You are about to upgrade *MinIO Server*, please confirm [y/N]: ")
		answer, e := bufio.NewReader(os.Stdin).ReadString('\n')
		fatalIf(probe.NewError(e), "Unable to parse user input.")
		answer = strings.TrimSpace(answer)
		if answer = strings.ToLower(answer); answer != "y" && answer != "yes" {
			fmt.Println("Upgrade aborted!")
			return nil
		}
	}

	// Update the specified MinIO server, optionally also
	// with the provided update URL.
	us, e := client.ServerUpdate(globalContext, madmin.ServerUpdateOpts{
		DryRun:    cmd.Bool("dry-run"),
		UpdateURL: updateURL,
	})
	fatalIf(probe.NewError(e), "Unable to update the server.")

	// Success..
	printMsg(serverUpdateMessage{
		Status:             "success",
		ServerURL:          aliasedURL,
		ServerUpdateStatus: us,
	})
	return nil
}
