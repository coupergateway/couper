#!/bin/bash
# Pre-commit hook for Claude Code: ensures go fmt and go vet pass before commit
# Exit codes:
#   0 - proceed with the command
#   2 - block the command (error message sent to Claude)

set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Only intercept git commit commands
if ! echo "$COMMAND" | grep -qE '^\s*git\s+commit'; then
  exit 0
fi

cd "$CLAUDE_PROJECT_DIR" || exit 0

# Check if there are staged Go files
STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM 2>/dev/null | grep '\.go$' || true)
if [ -z "$STAGED_GO_FILES" ]; then
  exit 0
fi

echo "Running pre-commit Go checks..." >&2

# Run go fmt on staged files
echo "Checking go fmt..." >&2
UNFORMATTED=$(gofmt -l $STAGED_GO_FILES 2>/dev/null || true)
if [ -n "$UNFORMATTED" ]; then
  echo "ERROR: The following files need formatting with 'go fmt':" >&2
  echo "$UNFORMATTED" >&2
  echo "" >&2
  echo "Run 'go fmt ./...' to fix, then 'git add' the files again before committing." >&2
  exit 2
fi

# Run go vet on the whole project, excluding fuzz folder
echo "Checking go vet..." >&2
VET_OUTPUT=$(go list ./... | grep -v 'fuzz\/.*' | xargs go vet 2>&1) || {
  echo "ERROR: go vet found issues:" >&2
  echo "$VET_OUTPUT" >&2
  echo "" >&2
  echo "Fix the issues above before committing." >&2
  exit 2
}

echo "Go checks passed." >&2
exit 0
