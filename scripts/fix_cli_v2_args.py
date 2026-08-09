#!/usr/bin/env python3
"""Fix urfave/cli/v2 Args API usages in cmd/."""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CMD = ROOT / "cmd"


def fix_args_api(text: str) -> str:
    text = text.replace("len(ctx.Args())", "ctx.Args().Len()")
    text = text.replace("range ctx.Args()", "range ctx.Args().Slice()")
    text = re.sub(r"\bargs := ctx\.Args\(\)", "args := ctx.Args().Slice()", text)
    text = re.sub(r"ctx\.Args\(\)(?!\.|\))", "ctx.Args().Slice()", text)
    return text


def main() -> None:
    for path in sorted(CMD.rglob("*.go")):
        original = path.read_text()
        updated = fix_args_api(original)
        if updated != original:
            path.write_text(updated)


if __name__ == "__main__":
    main()
