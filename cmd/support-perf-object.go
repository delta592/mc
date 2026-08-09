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
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/delta592/mc/pkg/probe"
	"github.com/dustin/go-humanize"
	"github.com/minio/madmin-go/v4"
	"github.com/urfave/cli/v3"
)

var adminSpeedtestCmd = &cli.Command{
	Name:               "speedtest",
	Usage:              "Run server side speed test",
	Action:             mainAdminSpeedtest,
	OnUsageError:       onUsageError,
	Before:             setGlobalsFromContext,
	HideHelpCommand:    true,
	Hidden:             true,
	CustomHelpTemplate: "Please use 'mc support perf'",
}

func mainAdminSpeedtest(_ context.Context, _ *cli.Command) error {
	deprecatedError("mc support perf")
	return nil
}

func mainAdminSpeedTestObject(_ context.Context, cmd *cli.Command, aliasedURL string, outCh chan<- PerfTestResult) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	duration, e := time.ParseDuration(cmd.String("duration"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse duration")
		return nil
	}
	if duration <= 0 {
		fatalIf(errInvalidArgument(), "duration cannot be 0 or negative")
		return nil
	}
	size, e := humanize.ParseBytes(cmd.String("size"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse object size")
		return nil
	}
	if size <= 0 {
		fatalIf(errInvalidArgument(), "size is expected to be more than 0 bytes")
		return nil
	}
	concurrent := cmd.Int("concurrent")
	if concurrent <= 0 {
		fatalIf(errInvalidArgument(), "concurrency cannot be '0' or negative")
		return nil
	}
	globalPerfTestVerbose = cmd.Bool("verbose")

	// Turn-off autotuning only when "concurrent" is specified
	// in all other scenarios keep auto-tuning on.
	autotune := !cmd.IsSet("concurrent")

	resultCh, e := client.Speedtest(ctxt, madmin.SpeedtestOpts{
		Size:        int(size),
		Duration:    duration,
		Concurrency: concurrent,
		Autotune:    autotune,
		Bucket:      cmd.String("bucket"), // This is a hidden flag.
		NoClear:     cmd.Bool("noclear"),
	})

	if globalJSON {
		if e != nil {
			printMsg(convertPerfResult(PerfTestResult{
				Type:  ObjectPerfTest,
				Err:   e.Error(),
				Final: true,
			}))
			return nil
		}

		var result madmin.SpeedTestResult
		for result = range resultCh {
			if result.Version == "" {
				continue
			}
		}

		printMsg(convertPerfResult(PerfTestResult{
			Type:         ObjectPerfTest,
			ObjectResult: &result,
			Final:        true,
		}))

		return nil
	}

	done := make(chan struct{})

	p := tea.NewProgram(initSpeedTestUI())
	go func() {
		if _, e := p.Run(); e != nil {
			os.Exit(1)
		}
		close(done)
	}()

	go func() {
		if e != nil {
			r := PerfTestResult{
				Type:  ObjectPerfTest,
				Err:   e.Error(),
				Final: true,
			}
			p.Send(r)
			if outCh != nil {
				outCh <- r
			}
			return
		}

		var result madmin.SpeedTestResult
		for result = range resultCh {
			p.Send(PerfTestResult{
				Type:         ObjectPerfTest,
				ObjectResult: &result,
			})
		}
		r := PerfTestResult{
			Type:         ObjectPerfTest,
			ObjectResult: &result,
			Final:        true,
		}
		p.Send(r)
		if outCh != nil {
			outCh <- r
		}
	}()

	<-done

	return nil
}
