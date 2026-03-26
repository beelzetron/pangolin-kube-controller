---
name: controller-change-review
description: Review and validate changes to the Kubernetes controller logic, including reconciliation loop, apply logic, transformation, and config handling. Use when modifying internal controller behavior or Kubernetes resource handling.
---

# Controller Change Review Skill

## When to use

Use this skill when changes affect:

- `internal/controller/`
- `internal/apply/`
- `internal/reconcile/`
- `internal/transform/`
- `internal/config/`
- `internal/httpserver/`
- `cmd/controller/`
- `deploy/`

---

## Core invariants (MUST NOT BREAK)

Always preserve:

- Reconciliation must be **idempotent**
- Controller must be **safe under retries**
- Backoff must be **bounded and non-aggressive**
- Leader election must not be bypassed
- Garbage collection must never delete unrelated resources
- Apply must remain **safe and scoped**
- Traefik CRDs must remain **compatible**

---

## Review checklist

### Reconciliation loop

- Does the loop remain safe under repeated execution?
- Are external calls (Pangolin API) guarded against failures?
- Is change detection (ETag/hash) preserved?

### Apply logic

- Are resources applied using server-side apply safely?
- Are field managers consistent?
- Are partial updates avoided?

### Garbage collection

- Only deletes resources owned by this controller?
- Label selectors correct?
- No risk of deleting shared/global resources?

### Transformation

- Traefik objects correctly generated?
- Routing, middleware, services preserved?
- No invalid CRDs produced?

### Config handling

- Defaults preserved?
- Env + file merging still correct?
- No breaking changes to config fields?

### HTTP server / observability

- Health and readiness endpoints still valid?
- Metrics preserved?

---

## Required validation

Run:

```bash
task test
task test:integration
go vet ./...
```

Optional deep validation:

```bash
task ci
```

---

## Expected output

* Summary of change impact
* Risks identified
* Required fixes (if any)
* Validation commands executed
