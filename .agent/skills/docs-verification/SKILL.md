---
name: docs-verification
description: Verify and update documentation against actual repository behavior, commands, and workflows. Use when editing README, docs, or changelog.
---

# Documentation Verification Skill

## When to use

- Editing `README.md`
- Editing `docs/`
- Updating `CHANGELOG.md`
- Updating `CONTRIBUTING.md`

---

## Rules

- Never invent commands
- Verify every command against repo
- Prefer Taskfile commands

---

## Validation commands

```bash
task build
task test
task ci
```

---

## Lint docs

```bash
markdownlint-cli2 "**/*.md"
```

---

## Checklist

* Commands exist?
* Flags correct?
* Paths correct?
* Matches workflows?

---

## Output

* Summary
* Files changed
* Commands verified
* Notes
