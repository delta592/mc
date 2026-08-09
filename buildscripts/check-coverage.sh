#!/usr/bin/env bash
#
# Enforces a minimum aggregate statement coverage over the first-party pkg/
# tree, and prints a per-package breakdown.
#
# Usage: check-coverage.sh [min-percent]   (default 80)
set -euo pipefail

min="${1:-80}"
profile="${COVERAGE_FILE:-coverage.out}"

if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

python3 - "$profile" "$min" <<'PY'
import sys
from collections import defaultdict

profile, min_pct = sys.argv[1], float(sys.argv[2])
pkg_stats = defaultdict(lambda: {"stmts": 0, "covered": 0})

with open(profile) as f:
    f.readline()  # mode: line
    for line in f:
        line = line.strip()
        if not line:
            continue
        loc, stmts, count = line.rsplit(" ", 2)
        stmts, count = int(stmts), int(count)
        path = loc.rsplit(":", 1)[0]
        if "/pkg/" not in path:
            continue
        # Group by the package directory the file lives in, so nested packages
        # such as pkg/textutil/ansi are reported separately from pkg/textutil.
        pkg = "pkg/" + path.split("/pkg/", 1)[1].rsplit("/", 1)[0]
        pkg_stats[pkg]["stmts"] += stmts
        if count > 0:
            pkg_stats[pkg]["covered"] += stmts

print(f"{'Package':<24}{'Coverage':>10}{'Statements':>12}")
print("-" * 46)
for pkg in sorted(pkg_stats):
    s = pkg_stats[pkg]
    pct = 100 * s["covered"] / s["stmts"] if s["stmts"] else 100.0
    print(f"{pkg:<24}{pct:9.1f}%{s['stmts']:12d}")

total_stmts = sum(s["stmts"] for s in pkg_stats.values())
total_cov = sum(s["covered"] for s in pkg_stats.values())
total_pct = 100 * total_cov / total_stmts if total_stmts else 100.0
print("-" * 46)
print(f"{'pkg total':<24}{total_pct:9.1f}%{total_stmts:12d}")

if total_pct + 1e-9 < min_pct:
    print(
        f"\nAggregate pkg coverage {total_pct:.1f}% is below the {min_pct:.0f}% threshold",
        file=sys.stderr,
    )
    sys.exit(1)
PY
