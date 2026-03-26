---
name: Feature request
about: Suggest an improvement or new capability
labels: [enhancement]
---

# Description

Describe the feature clearly.

---

## Problem Statement

What limitation exists today?

Tie it to:

* Reconciliation behavior
* Config transformation
* Observability
* Kubernetes integration

---

## Proposed Solution

Describe how it should work:

* API changes
* Config options (env / YAML)
* Controller behavior
* Example:

```yaml
FEATURE_FLAG_X: true
```

---

## Impact on Architecture

Which components are affected?

* [ ] controller loop
* [ ] transform layer
* [ ] apply (SSA)
* [ ] garbage collection
* [ ] observability
* [ ] HTTP server
* [ ] config system

---

## Testing Strategy

How should this be tested?

* Unit tests
* Integration tests (`envtest`)
* E2E tests

---

## Alternatives Considered

Other approaches you evaluated.

---

## Risks / Trade-offs

* Performance impact?
* Breaking changes?
* Increased API calls?

---

## Additional Context

Anything else (diagrams, references, etc.)
