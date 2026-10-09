#!/usr/bin/env bash
# Prints the changelog "Dependencies" entries for the go module bumps merged into a branch since its last release.
# Each module is listed once with its net change. A transitive module is listed only when a PR title names it.
# Usage: scripts/changelog-deps.sh [base-branch] [since-tag]
set -euo pipefail

base=${1:-main}
since_tag=${2:-$(git describe --tags --abbrev=0 "origin/$base")}
since=$(gh release view "$since_tag" --json publishedAt --jq .publishedAt)

gh pr list --base "$base" --state merged --limit 200 --json number,url,title,mergedAt,mergeCommit |
jq -r --arg since "$since" '[.[] | select(.mergedAt >= $since)] | sort_by(.mergedAt) | .[] | "\(.number) \(.url) \(.mergeCommit.oid) \(.title)"' |
while read -r number url oid title; do
  printf 'pr %s %s %s\n' "$number" "$url" "$title"
  git diff "$oid^" "$oid" -- go.mod
done |
awk '
  $1 == "pr" {
    link = "[#" $2 "](" $3 ")"
    title = $0
    sub(/^pr [^ ]+ [^ ]+ ?/, "", title)
    title = " " title " "
    next
  }
  $1 == "-go" { if (oldgo == "") oldgo = $2 }
  $1 == "+go" { newgo = $2; golinks = golinks (golinks ? ", " : "") link }
  $1 == "-" || $1 == "+" {
    p = $2
    if (!(p in seen)) { seen[p] = ++n; path[n] = p }
    if ($1 == "-") { if (!(p in old)) old[p] = substr($3, 2); next }
    new[p] = substr($3, 2)
    indirect[p] = ($0 ~ /\/\/ indirect/)
    if (index(title, " " p " ")) named[p] = 1
    if (last[p] != link) { links[p] = links[p] (links[p] ? ", " : "") link; last[p] = link }
  }
  END {
    if (newgo != "" && newgo != oldgo) printf "  * build with go %s (%s)\n", newgo, golinks
    for (i = 1; i <= n; i++) {
      p = path[i]
      if (!(p in old) || !(p in new) || old[p] == new[p]) continue
      if (indirect[p] && !named[p]) continue
      printf "  * `%s` %s → %s%s (%s)\n", p, old[p], new[p], indirect[p] ? ", transitive" : "", links[p]
    }
  }'
