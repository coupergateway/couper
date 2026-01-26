---
name: init-context
description: Load full project context by reading all documentation linked from CLAUDE.md
allowed-tools: Read, Grep, Glob
---

# Init Context

1. Parse `CLAUDE.md` and find all markdown links to local documentation files (paths like `docs/*.md` or relative `.md` links)
2. Read each linked documentation file fully
3. Confirm context is loaded with a brief summary of key concepts
