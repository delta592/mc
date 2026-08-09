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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/delta592/mc/pkg/probe"
	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7/pkg/set"
	"github.com/minio/pkg/v3/console"
	"github.com/minio/pkg/v3/env"
	"github.com/minio/pkg/v3/trie"
	"github.com/minio/pkg/v3/words"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// global flags for mc.
var mcFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:  "autocompletion",
		Usage: "install auto-completion for your shell",
	},
}

// Help template for mc
var mcHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} {{if .VisibleFlags}}[FLAGS] {{end}}COMMAND{{if .VisibleFlags}} [COMMAND FLAGS | -h]{{end}} [ARGUMENTS...]

COMMANDS:
  {{range .VisibleCommands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .VisibleFlags}}
GLOBAL FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
TIP:
  Use '{{.Name}} --autocompletion' to enable shell autocompletion

COPYRIGHT:
  Copyright (c) 2015-` + CopyrightYear + ` MinIO, Inc.

LICENSE:
  GNU AGPLv3 <https://www.gnu.org/licenses/agpl-3.0.html>
`

func init() {
	if env.IsSet(mcEnvConfigFile) {
		configFile := env.Get(mcEnvConfigFile, "")
		fatalIf(readAliasesFromFile(configFile).Trace(configFile), "Unable to parse "+configFile)
	}
	if runtime.GOOS == "windows" {
		if startedByExplorer() {
			fmt.Printf("Don't double-click %s\n", os.Args[0])
			fmt.Println("You need to open cmd.exe/PowerShell and run it from the command line")
			fmt.Println("Press the Enter Key to Exit")
			fmt.Scanln()
			os.Exit(1)
		}
	}
}

// Main starts mc application
func Main(args []string) error {
	// ``MC_PROFILER`` supported options are [cpu, mem, block, goroutine].
	if p := os.Getenv("MC_PROFILER"); p != "" {
		profilers := strings.Split(p, ",")
		if e := enableProfilers(mustGetProfileDir(), profilers); e != nil {
			console.Fatal(e)
		}
	}

	probe.Init() // Set project's root source path.
	probe.SetAppInfo("Release-Tag", ReleaseTag)
	probe.SetAppInfo("Commit", ShortCommitID)

	// Fetch terminal size, if not available, automatically
	// set globalQuiet to true on non-window.
	if w, h, e := term.GetSize(int(os.Stdout.Fd())); e != nil {
		globalQuiet = runtime.GOOS != "windows"
	} else {
		globalTermWidth, globalTermHeight = w, h
	}

	// Set the mc app name.
	appName := filepath.Base(args[0])
	if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(appName), ".exe") {
		// Trim ".exe" from Windows executable.
		appName = appName[:strings.LastIndex(appName, ".")]
	}

	// Monitor OS exit signals and cancel the global context in such case
	go trapSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

	// Run the app
	return registerApp(appName).Run(globalContext, args)
}

func flagValue(f cli.Flag) reflect.Value {
	fv := reflect.ValueOf(f)
	for fv.Kind() == reflect.Pointer {
		fv = reflect.Indirect(fv)
	}
	return fv
}

func visibleFlags(fl []cli.Flag) []cli.Flag {
	visible := []cli.Flag{}
	for _, flag := range fl {
		field := flagValue(flag).FieldByName("Hidden")
		if !field.IsValid() || !field.Bool() {
			visible = append(visible, flag)
		}
	}
	return visible
}

// Function invoked when invalid flag is passed
func onUsageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	type subCommandHelp struct {
		flagName string
		usage    string
	}

	// Calculate the maximum width of the flag name field
	// for a good looking printing
	vflags := visibleFlags(cmd.Flags)
	help := make([]subCommandHelp, len(vflags))
	maxWidth := 0
	for i, f := range vflags {
		s := strings.Split(f.String(), "\t")
		if len(s[0]) > maxWidth {
			maxWidth = len(s[0])
		}

		help[i] = subCommandHelp{flagName: s[0], usage: s[1]}
	}
	maxWidth += 2

	var errMsg strings.Builder

	// Do the good-looking printing now
	fmt.Fprintln(&errMsg, "Invalid command usage,", err.Error())
	if len(help) > 0 {
		fmt.Fprintln(&errMsg, "\nSUPPORTED FLAGS:")
		for _, h := range help {
			spaces := string(bytes.Repeat([]byte{' '}, maxWidth-len(h.flagName)))
			fmt.Fprintf(&errMsg, "   %s%s%s\n", h.flagName, spaces, h.usage)
		}
	}
	console.Fatal(errMsg.String())
	return err
}

// Function invoked when invalid command is passed.
func commandNotFound(cmd *cli.Command, cmds []*cli.Command) {
	command := cmd.Args().First()
	if command == "" {
		cli.ShowCommandHelp(globalContext, cmd, command)
		return
	}
	var msg strings.Builder
	fmt.Fprintf(&msg, "`%s` is not a recognized command. Get help using `--help` flag.", command)
	commandsTree := trie.NewTrie()
	for _, cmd := range cmds {
		commandsTree.Insert(cmd.Name)
	}
	closestCommands := findClosestCommands(commandsTree, command)
	if len(closestCommands) > 0 {
		msg.WriteString("\n\nDid you mean one of these?\n")
		if len(closestCommands) == 1 {
			cmd := closestCommands[0]
			fmt.Fprintf(&msg, "        `%s`", cmd)
		} else {
			for _, cmd := range closestCommands {
				fmt.Fprintf(&msg, "        `%s`\n", cmd)
			}
		}
	}
	fatalIf(errDummy().Trace(), msg.String())
}

// Check for sane config environment early on and gracefully report.
func checkConfig() {
	// Refresh the config once.
	loadMcConfig = loadMcConfigFactory()
	// Ensures config file is sane.
	config, err := loadMcConfig()
	// Verify if the path is accesible before validating the config
	fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to access configuration file.")

	// Validate and print error messges
	ok, errMsgs := validateConfigFile(config)
	if !ok {
		var errorMsg bytes.Buffer
		for index, errMsg := range errMsgs {
			// Print atmost 10 errors
			if index > 10 {
				break
			}
			errorMsg.WriteString(errMsg + "\n")
		}
		console.Fatal(errorMsg.String())
	}
}

func migrate() {
	// Fix broken config files if any.
	fixConfig()

	// Migrate config files if any.
	migrateConfig()

	// Migrate shared urls if any.
	migrateShare()
}

// initMC - initialize 'mc'.
func initMC() {
	// Check if mc config exists.
	if !isMcConfigExists() {
		err := saveMcConfig(newMcConfig())
		fatalIf(err.Trace(), "Unable to save new mc config.")

		if !globalQuiet && !globalJSON {
			console.Infoln("Configuration written to `" + mustGetMcConfigPath() + "`. Please update your access credentials.")
		}
	}

	// Check if mc share directory exists.
	if !isShareDirExists() {
		initShareConfig()
	}

	// Check if certs dir exists
	if !isCertsDirExists() {
		fatalIf(createCertsDir().Trace(), "Unable to create `CAs` directory.")
	}

	// Check if CAs dir exists
	if !isCAsDirExists() {
		fatalIf(createCAsDir().Trace(), "Unable to create `CAs` directory.")
	}

	// Load all authority certificates present in CAs dir
	loadRootCAs()
}

func getShellName() (string, bool) {
	shellName := os.Getenv("SHELL")
	if shellName != "" || runtime.GOOS == "windows" {
		return strings.ToLower(filepath.Base(shellName)), true
	}

	ppid := os.Getppid()
	cmd := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=")
	ppName, err := cmd.Output()
	if err != nil {
		fatalIf(probe.NewError(err), "Failed to enable autocompletion. Cannot determine shell type and "+
			"no SHELL environment variable found")
	}
	shellName = strings.TrimSpace(string(ppName))
	return strings.ToLower(filepath.Base(shellName)), false
}

func installAutoCompletion() {
	if runtime.GOOS == "windows" {
		console.Infoln("autocompletion feature is not available for this operating system")
		return
	}

	shellName, ok := getShellName()
	if !ok {
		console.Infoln("No 'SHELL' env var. Your shell is auto determined as '" + shellName + "'.")
	} else {
		console.Infoln("Your shell is set to '" + shellName + "', by env var 'SHELL'.")
	}

	supportedShellsSet := set.CreateStringSet("bash", "zsh", "fish")
	if !supportedShellsSet.Contains(shellName) {
		fatalIf(probe.NewError(errors.New("")),
			"'"+shellName+"' is not a supported shell. "+
				"Supported shells are: bash, zsh, fish")
	}

	if err := installShellCompletion(filepath.Base(os.Args[0])); err != nil {
		if isShellCompletionInstalled(filepath.Base(os.Args[0])) || isShellCompletionInstalled("mc") {
			console.Infoln("autocompletion is enabled.")
			return
		}
		fatalIf(probe.NewError(err), "Unable to install auto-completion.")
		return
	}
	console.Infoln("enabled autocompletion in your '" + shellName + "' rc file. Please restart your shell.")
}

func registerBefore(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	deprecatedFlagsWarning(cmd)

	if cmd.IsSet("config-dir") {
		// Set the config directory.
		setMcConfigDir(cmd.String("config-dir"))
	}

	// Set global flags.
	if _, err := setGlobalsFromContext(ctx, cmd); err != nil {
		return ctx, err
	}

	// Migrate any old version of config / state files to newer format.
	migrate()

	// Initialize default config files.
	initMC()

	// Check if config can be read.
	checkConfig()

	return ctx, nil
}

// findClosestCommands to match a given string with commands trie tree.
func findClosestCommands(commandsTree *trie.Trie, command string) []string {
	closestCommands := commandsTree.PrefixMatch(command)
	sort.Strings(closestCommands)
	// Suggest other close commands - allow missed, wrongly added and even transposed characters
	for _, value := range commandsTree.Walk(commandsTree.Root()) {
		if sort.SearchStrings(closestCommands, value) < len(closestCommands) {
			continue
		}
		// 2 is arbitrary and represents the max allowed number of typed errors
		if words.DamerauLevenshteinDistance(command, value) < 2 {
			closestCommands = append(closestCommands, value)
		}
	}
	return closestCommands
}

// Check for updates and print a notification message
func checkUpdate(_ context.Context, cmd *cli.Command) {
	// Do not print update messages, if quiet flag is set.
	if !cmd.Bool("quiet") {
		// Its OK to ignore any errors during doUpdate() here.
		if updateMsg, _, currentReleaseTime, latestReleaseTime, _, err := getUpdateInfo("", 2*time.Second); err == nil {
			printMsg(updateMessage{
				Status:  "success",
				Message: updateMsg,
			})
		} else {
			printMsg(updateMessage{
				Status:  "success",
				Message: prepareUpdateMessage("Run `mc update`", latestReleaseTime.Sub(currentReleaseTime)),
			})
		}
	}
}

var appCmds = []*cli.Command{
	aliasCmd,
	adminCmd,
	anonymousCmd,
	batchCmd,
	cpCmd,
	catCmd,
	corsCmd,
	diffCmd,
	duCmd,
	encryptCmd,
	eventCmd,
	findCmd,
	getCmd,
	headCmd,
	ilmCmd,
	idpCmd,
	licenseCmd,
	legalHoldCmd,
	lsCmd,
	mbCmd,
	mvCmd,
	mirrorCmd,
	odCmd,
	pingCmd,
	policyCmd,
	pipeCmd,
	putCmd,
	quotaCmd,
	rmCmd,
	retentionCmd,
	rbCmd,
	replicateCmd,
	readyCmd,
	sqlCmd,
	statCmd,
	supportCmd,
	shareCmd,
	treeCmd,
	tagCmd,
	undoCmd,
	updateCmd,
	versionCmd,
	watchCmd,
}

func printMCVersion(c *cli.Command) {
	root := c.Root()
	fmt.Fprintf(root.Writer, "%s version %s (commit-id=%s)\n", root.Name, root.Version, CommitID)
	fmt.Fprintf(root.Writer, "Runtime: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(root.Writer, "Copyright (c) 2015-%s MinIO, Inc.\n", CopyrightYear)
	fmt.Fprintf(root.Writer, "License GNU AGPLv3 <https://www.gnu.org/licenses/agpl-3.0.html>\n")
}

func registerApp(name string) *cli.Command {
	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "show help",
	}

	// Override default cli version printer
	cli.VersionPrinter = printMCVersion

	app := &cli.Command{
		Name:                          name,
		Before:                        registerBefore,
		HideHelpCommand:               true,
		Usage:                         "MinIO Client for object storage and filesystems.",
		Commands:                      appCmds,
		Authors:                       []any{mail.Address{Name: "MinIO, Inc."}},
		Version:                       ReleaseTag,
		Flags:                         append(mcFlags, globalFlags...),
		CustomRootCommandHelpTemplate: mcHelpTemplate,
		EnableShellCompletion:         true,
		OnUsageError:                  onUsageError,
		After: func(_ context.Context, _ *cli.Command) error {
			globalExpiringCerts.Range(func(k, v any) bool {
				host := k.(string)
				expires := v.(time.Time)
				fmt.Fprintf(os.Stderr, "\n")
				fmt.Fprintf(os.Stderr, "== WARN: `%s` certificate will expire in %s. Renew soon to avoid outage.\n", host, expires)
				fmt.Fprintf(os.Stderr, "\n")
				return true
			})
			return nil
		},
		Writer: os.Stdout,
	}
	app.Action = func(_ context.Context, cmd *cli.Command) error {
		mcEnable := env.Get("MC_UPDATE", madmin.EnableOn)
		minioEnable := env.Get("MINIO_UPDATE", madmin.EnableOn)

		if strings.HasPrefix(ReleaseTag, "RELEASE.") && (mcEnable == madmin.EnableOn || minioEnable == madmin.EnableOn) {
			// Check for new updates from dl.min.io.
			checkUpdate(globalContext, cmd)
		}

		if cmd.Bool("autocompletion") {
			// Install shell completions
			installAutoCompletion()
			return nil
		}

		if cmd.Args().First() == "" {
			showAppHelpAndExit(cmd)
		}

		commandNotFound(cmd, app.Commands)
		return exitStatus(globalErrorExitStatus)
	}

	wireShellCompletions(app.Commands, "")

	return app
}

// mustGetProfilePath must get location that the profile will be written to.
func mustGetProfileDir() string {
	return filepath.Join(mustGetMcConfigDir(), globalProfileDir)
}

func showCommandHelpAndExit(cmd *cli.Command, code int) {
	cli.ShowCommandHelpAndExit(globalContext, cmd, cmd.Name, code)
}

func showAppHelpAndExit(cmd *cli.Command) {
	cli.ShowRootCommandHelpAndExit(cmd.Root(), globalErrorExitStatus)
}
