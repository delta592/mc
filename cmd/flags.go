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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/delta592/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/urfave/cli/v2"
)

const envPrefix = "MC_"

// Collection of mc flags currently supported
var globalFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "config-dir",
		Aliases: []string{"C"},
		Value:   mustGetMcConfigDir(),
		Usage:   "path to configuration folder",
		EnvVars: []string{envPrefix + "CONFIG_DIR"},
	},
	&cli.BoolFlag{
		Name:    "quiet",
		Aliases: []string{"q"},
		Usage:   "disable progress bar display",
		EnvVars: []string{envPrefix + "QUIET"},
	},
	&cli.BoolFlag{
		Name:    "no-color",
		Usage:   "disable color theme",
		EnvVars: []string{envPrefix + "NO_COLOR"},
	},
	&cli.BoolFlag{
		Name:    "json",
		Usage:   "enable JSON lines formatted output",
		EnvVars: []string{envPrefix + "JSON"},
	},
	&cli.BoolFlag{
		Name:    "debug",
		Usage:   "enable debug output",
		EnvVars: []string{envPrefix + "DEBUG"},
	},
	&cli.StringSliceFlag{
		Name:    "resolve",
		Usage:   "resolves HOST[:PORT] to an IP address. Example: minio.local:9000=10.10.75.1",
		EnvVars: []string{envPrefix + "RESOLVE"},
	},
	&cli.BoolFlag{
		Name:    "insecure",
		Usage:   "disable SSL certificate verification",
		EnvVars: []string{envPrefix + "INSECURE"},
	},
	&cli.StringFlag{
		Name:    "limit-upload",
		Usage:   "limits uploads to a maximum rate in KiB/s, MiB/s, GiB/s. (default: unlimited)",
		EnvVars: []string{envPrefix + "LIMIT_UPLOAD"},
	},
	&cli.StringFlag{
		Name:    "limit-download",
		Usage:   "limits downloads to a maximum rate in KiB/s, MiB/s, GiB/s. (default: unlimited)",
		EnvVars: []string{envPrefix + "LIMIT_DOWNLOAD"},
	},
	&cli.DurationFlag{
		Name:   "conn-read-deadline",
		Usage:  "custom connection READ deadline",
		Hidden: true,
		Value:  10 * time.Minute,
	},
	&cli.DurationFlag{
		Name:   "conn-write-deadline",
		Usage:  "custom connection WRITE deadline",
		Hidden: true,
		Value:  10 * time.Minute,
	},
	&cli.StringSliceFlag{
		Name:    "custom-header",
		Aliases: []string{"H"},
		Usage:   "add custom HTTP header to the request. 'key:value' format.",
	},
}

// bundled encryption flags
var encFlags = []cli.Flag{
	encCFlag,
	encKSMFlag,
	encS3Flag,
}

var encCFlag = &cli.StringSliceFlag{
	Name:  "enc-c",
	Usage: "encrypt/decrypt objects using client provided keys. (multiple keys can be provided) Formats: RawBase64 or Hex.",
}

var encKSMFlag = &cli.StringSliceFlag{
	Name:    "enc-kms",
	Usage:   "encrypt/decrypt objects using specific server-side encryption keys. (multiple keys can be provided)",
	EnvVars: []string{envPrefix + "ENC_KMS"},
}

var encS3Flag = &cli.StringSliceFlag{
	Name:    "enc-s3",
	Usage:   "encrypt/decrypt objects using server-side default keys and configurations. (multiple keys can be provided).",
	EnvVars: []string{envPrefix + "ENC_S3"},
}

var checksumFlag = &cli.StringFlag{
	Name:  "checksum",
	Usage: "Add checksum to uploaded object. Values: CRC64NVME, CRC32, CRC32C, SHA1 or SHA256. Requires server trailing headers (AWS, MinIO)",
	Value: "",
}

func parseChecksum(ctx *cli.Context) (useMD5 bool, ct minio.ChecksumType) {
	useMD5 = ctx.Bool("md5")
	if cs := ctx.String("checksum"); cs != "" {
		switch strings.ToUpper(cs) {
		case "CRC32":
			ct = minio.ChecksumCRC32
		case "CRC32C":
			ct = minio.ChecksumCRC32C
		case "CRC32-FO":
			ct = minio.ChecksumFullObjectCRC32
		case "CRC32C-FO":
			ct = minio.ChecksumFullObjectCRC32C
		case "SHA1":
			ct = minio.ChecksumSHA1
		case "SHA256":
			ct = minio.ChecksumSHA256
		case "CRC64N", "CRC64NVME":
			ct = minio.ChecksumCRC64NVME
		case "MD5":
			useMD5 = true
		default:
			err := fmt.Errorf("unknown checksum type: %s. Should be one of CRC64NVME, MD5, CRC32, CRC32C, SHA1 or SHA256", cs)
			fatalIf(probe.NewError(err), "")
		}
		if ct.IsSet() {
			useTrailingHeaders.Store(true)
			if useMD5 {
				err := errors.New("cannot combine MD5 with checksum")
				fatalIf(probe.NewError(err), "")
			}
		}
	}
	return
}
