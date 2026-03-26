---
name: envtest-integration
description: Setup, run, and debug Kubernetes integration tests using envtest. Use when working with test/integration or Kubernetes client behavior.
---

# Envtest Integration Skill

## When to use

- Changes in `test/integration/`
- Changes in Kubernetes client logic (`internal/kube/`)
- CRD-related changes
- Controller behavior affecting API interaction

---

## Setup

Ensure envtest is installed:

```bash
setup-envtest use -p env 1.29.x
```

---

## Run tests

```bash
task test:integration
```

or:

```bash
go test -tags=integration ./test/integration
```

---

## Debugging checklist

* CRDs correctly loaded?
* API server started?
* Scheme registration correct?
* Resources created before assertions?
* Context timeouts correct?

---

## Common issues

* Missing CRDs → ensure testdata paths correct
* Race conditions → check goroutines and async logic
* Timeouts → increase test context duration

---

## Validation

```bash
task test
task test:integration
```

---

## Output format

* What failed
* Root cause
* Fix suggestion
* Commands executed
