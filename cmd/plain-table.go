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
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

type plainTableConfig struct {
	align            tw.Align
	columnPadding    string
	autoFormatHeader bool
}

func newPlainTable(w io.Writer, cfg plainTableConfig) *tablewriter.Table {
	if cfg.columnPadding == "" {
		cfg.columnPadding = "\t"
	}

	autoFormat := tw.Off
	if cfg.autoFormatHeader {
		autoFormat = tw.On
	}

	cellPadding := tw.CellPadding{
		Global: tw.Padding{Right: cfg.columnPadding, Overwrite: true},
	}

	return tablewriter.NewTable(w,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Symbols: tw.NewSymbols(tw.StyleNone),
			Settings: tw.Settings{
				Lines:      tw.LinesNone,
				Separators: tw.SeparatorsNone,
			},
		})),
		tablewriter.WithHeaderAlignment(cfg.align),
		tablewriter.WithRowAlignment(cfg.align),
		tablewriter.WithHeaderAutoFormat(autoFormat),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithTrimSpace(tw.Off),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{Padding: cellPadding},
			Row:    tw.CellConfig{Padding: cellPadding},
		}),
	)
}

func renderPlainTable(table *tablewriter.Table) {
	_ = table.Render()
}
