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

	json "github.com/delta592/mc/pkg/colorjson"
	"github.com/delta592/mc/pkg/probe"
	"github.com/fatih/color"
	"github.com/minio/pkg/v3/console"
	"github.com/urfave/cli/v3"
)

var encryptSetCmd = &cli.Command{
	Name:         "set",
	Usage:        "set encryption config",
	Action:       mainEncryptSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
   
USAGE:
  {{.HelpName}} <sse-type> [<key-id>] TARGET
   
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable SSE-KMS auto encryption with KMS key on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} sse-kms my-minio-key myminio/mybucket

  2. Enable SSE-KMS auto encryption with KMS key on bucket "mybucket" for alias "s3".
     {{.Prompt}} {{.HelpName}} sse-kms arn:aws:kms:us-east-1:xxx:key/xxx s3/mybucket  
`,
}

// checkEncryptSetSyntax - validate all the passed arguments
func checkEncryptSetSyntax(cmd *cli.Command) {
	if cmd.Args().Len() < 2 || cmd.Args().Len() > 3 {
		showCommandHelpAndExit(cmd, 1) // last argument is exit code
	}
}

type encryptSetMessage struct {
	Op         string `json:"op"`
	Status     string `json:"status"`
	URL        string `json:"url"`
	Encryption struct {
		Algorithm string `json:"algorithm,omitempty"`
		KeyID     string `json:"keyId,omitempty"`
	} `json:"encryption"`
}

func (v encryptSetMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v encryptSetMessage) String() string {
	return console.Colorize("encryptSetMessage", fmt.Sprintf("Auto encryption configuration has been set successfully for %s", v.URL))
}

func mainEncryptSet(_ context.Context, cmd *cli.Command) error {
	ctx, cancelencryptSet := context.WithCancel(globalContext)
	defer cancelencryptSet()

	console.SetColor("encryptSetMessage", color.New(color.FgGreen))

	checkEncryptSetSyntax(cmd)

	// Get the alias parameter from cli
	args := cmd.Args()
	argsSlice := args.Slice()
	aliasedURL := args.Get(args.Len() - 1)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	var algorithm, keyID string
	switch args.Len() {
	case 3:
		algorithm = strings.ToLower(argsSlice[0])
		keyID = argsSlice[1]
	case 2:
		algorithm = strings.ToLower(argsSlice[0])
	}
	if algorithm != "sse-s3" && algorithm != "sse-kms" {
		fatalIf(probe.NewError(fmt.Errorf("Unknown argument `%s` passed", algorithm)), "Invalid encryption algorithm")
	}
	fatalIf(client.SetEncryption(ctx, algorithm, keyID), "Unable to enable auto encryption")
	msg := encryptSetMessage{
		Op:     cmd.Name,
		Status: "success",
		URL:    aliasedURL,
	}
	msg.Encryption.Algorithm = algorithm
	printMsg(msg)
	return nil
}
