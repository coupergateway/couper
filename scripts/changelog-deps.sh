#!/usr/bin/env bash
# Prints the changelog "Dependencies" entries for the go module bumps merged into a branch since its last release.
# Usage: scripts/changelog-deps.sh [base-branch] [since-tag]
set -euo pipefail

base=${1:-main}
since_tag=${2:-$(git describe --tags --abbrev=0 "origin/$base")}
since=$(gh release view "$since_tag" --json publishedAt --jq .publishedAt)

gh pr list --base "$base" --state merged --limit 200 --json number,url,mergedAt,mergeCommit |
jq -r --arg since "$since" '.[] | select(.mergedAt >= $since) | "\(.number) \(.url) \(.mergeCommit.oid)"' |
while read -r number url oid; do
  git diff "$oid^" "$oid" -- go.mod | awk -v link="([#$number]($url))" '
    $1 == "-go" { oldgo = $2 }
    $1 == "+go" { newgo = $2 }
    $1 == "-" || $1 == "+" {
      if (!($2 in seen)) { seen[$2] = ++n; path[n] = $2 }
      ver[$1, $2] = substr($3, 2)
      if ($1 == "+" && $0 ~ /\/\/ indirect/) transitive[$2] = ", transitive"
    }
    END {
      if (newgo != "" && newgo != oldgo) printf "  * build with go %s %s\n", newgo, link
      for (i = 1; i <= n; i++) {
        p = path[i]
        if (("-", p) in ver && ("+", p) in ver) printf "  * `%s` %s → %s%s %s\n", p, ver["-", p], ver["+", p], transitive[p], link
      }
    }'
done
