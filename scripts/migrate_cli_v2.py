#!/usr/bin/env python3
"""Migrate github.com/minio/cli (v1) to github.com/urfave/cli/v2."""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CMD = ROOT / "cmd"
ILM = ROOT / "cmd" / "ilm"

FLAG_TYPES = (
    "BoolFlag",
    "StringFlag",
    "IntFlag",
    "Int64Flag",
    "UintFlag",
    "Uint64Flag",
    "Float64Flag",
    "DurationFlag",
    "StringSliceFlag",
    "IntSliceFlag",
    "UintSliceFlag",
    "Float64SliceFlag",
    "GenericFlag",
)

GLOBAL_REPLACEMENTS = [
    (r"ctx\.GlobalBool\(", "ctx.Bool("),
    (r"ctx\.GlobalIsSet\(", "ctx.IsSet("),
    (r"ctx\.GlobalString\(", "ctx.String("),
    (r"ctx\.GlobalInt\(", "ctx.Int("),
    (r"ctx\.GlobalUint\(", "ctx.Uint("),
    (r"ctx\.GlobalDuration\(", "ctx.Duration("),
    (r"ctx\.GlobalFloat64\(", "ctx.Float64("),
    (r"ctx\.GlobalStringSlice\(", "ctx.StringSlice("),
]

NAME_ALIAS_RE = re.compile(
    r'^(?P<indent>\s*)Name:\s*"(?P<name>[^",]+),\s*(?P<alias>[^"]+)"\s*,?\s*$'
)


def split_flag_names(content: str) -> str:
    lines = content.splitlines(keepends=True)
    out = []
    i = 0
    while i < len(lines):
        m = NAME_ALIAS_RE.match(lines[i].rstrip("\n"))
        if m:
            indent = m.group("indent")
            name = m.group("name").strip()
            alias = m.group("alias").strip()
            out.append(f'{indent}Name: "{name}",\n')
            out.append(f'{indent}Aliases: []string{{"{alias}"}},\n')
            i += 1
            continue
        out.append(lines[i])
        i += 1
    return "".join(out)


def migrate_file(path: Path) -> bool:
    text = path.read_text()
    if "github.com/minio/cli" not in text:
        return False

    text = text.replace('"github.com/minio/cli"', '"github.com/urfave/cli/v2"')

    for flag in FLAG_TYPES:
        text = text.replace(f"cli.{flag}{{", f"&cli.{flag}{{")

    text = re.sub(
        r"EnvVar:\s*(" + re.escape("envPrefix") + r"\s*\+\s*[^,\n]+)",
        r"EnvVars: []string{\1}",
        text,
    )
    text = re.sub(r'EnvVar:\s*"([^"]+)"', r'EnvVars: []string{"\1"}', text)

    text = text.replace("var ", "var ")  # noop anchor
    text = re.sub(
        r"=\s*cli\.Command\{",
        "= &cli.Command{",
        text,
    )
    text = re.sub(
        r"\[\]cli\.Command\{",
        "[]*cli.Command{",
        text,
    )
    text = re.sub(
        r"func\s+\w+\(cli\.Command,",
        lambda m: m.group(0).replace("cli.Command", "*cli.Command"),
        text,
    )
    text = re.sub(
        r"func\s+\w+\([^)]*cli\.Command[^)]*\)",
        lambda m: m.group(0).replace("cli.Command", "*cli.Command"),
        text,
    )
    text = re.sub(
        r"cmds \[\]cli\.Command",
        "cmds []*cli.Command",
        text,
    )

    for pat, repl in GLOBAL_REPLACEMENTS:
        text = re.sub(pat, repl, text)

    text = split_flag_names(text)

    if path.name == "main.go":
        text = text.replace(
            "cli.HelpFlag = &cli.BoolFlag{\n\t\tName:  \"help\",\n\t\tUsage: \"show help\",\n\t}",
            "cli.HelpFlag = &cli.BoolFlag{\n\t\tName:    \"help\",\n\t\tAliases: []string{\"h\"},\n\t\tUsage:   \"show help\",\n\t}",
        )
        # handle pre-migration form too
        text = re.sub(
            r'cli\.HelpFlag = &cli\.BoolFlag\{\s*Name:\s*"help, h",\s*Usage:\s*"show help",\s*\}',
            'cli.HelpFlag = &cli.BoolFlag{\n\t\tName:    "help",\n\t\tAliases: []string{"h"},\n\t\tUsage:   "show help",\n\t}',
            text,
        )

    path.write_text(text)
    return True


def main() -> int:
    changed = 0
    for directory in (CMD,):
        for path in sorted(directory.rglob("*.go")):
            if migrate_file(path):
                changed += 1
                print(path.relative_to(ROOT))
    print(f"migrated {changed} files")
    return 0


if __name__ == "__main__":
    sys.exit(main())
