# Summary

Short description of what this PR does.

Fixes: <issue-number> (if applicable)

---

## Type of Change

- [ ] Feature
- [ ] Bug fix
- [ ] Refactor
- [ ] Documentation
- [ ] CI/CD
- [ ] Test improvement
- [ ] Other (describe below)

---

## Context & Motivation

Why is this change needed?

- What problem does it solve?
- Is this a regression or new capability?

---

## Affected Components

Select all that apply:

- [ ] controller (reconciliation loop)
- [ ] fetch (Pangolin API interaction)
- [ ] transform (routing, sanitize, protocol)
- [ ] apply (Server-Side-Apply logic)
- [ ] garbage collection
- [ ] kube client / resource handling
- [ ] config system
- [ ] HTTP server (/metrics, /healthz)
- [ ] observability (metrics/logging/tracing)
- [ ] leader election / HA
- [ ] CI/CD / build system
- [ ] documentation

---

## Behavior Changes

Does this PR change runtime behavior?

- [ ] Reconciliation logic
- [ ] Resource generation (Traefik CRDs)
- [ ] Pangolin API interaction
- [ ] Garbage collection behavior
- [ ] Metrics / observability
- [ ] Configuration interface (env/config file)

If yes, describe in detail:

---

## Implementation Details

Explain key design decisions:

- Why this approach?
- Any alternatives rejected?
- Trade-offs?

---

## Testing

### Automated

- [ ] Unit tests added/updated
- [ ] Integration tests added/updated (`envtest`)
- [ ] E2E tests added/updated

Coverage:

- [ ] Coverage ≥ 75%
- [ ] Edge cases covered (errors, retries, empty configs, etc.)

---

### Manual Testing

Describe how this was validated:

```bash
# Example:
kubectl apply -f deploy.yaml
kubectl logs -f deployment/pangolin-controller
```

Test scenarios:

- [ ] Config change detected
- [ ] No-change (ETag / 304 handling)
- [ ] Error handling + backoff
- [ ] Garbage collection correctness
- [ ] Leader election (if applicable)

---

## Observability Impact

- [ ] No changes
- [ ] New metrics added
- [ ] Existing metrics modified
- [ ] Logging improved

Details:

---

## Security Considerations

- [ ] No impact
- [ ] TLS/mTLS handling affected
- [ ] API authentication affected
- [ ] Sensitive data handling reviewed

Explain if relevant:

---

## Performance Impact

- [ ] No impact
- [ ] Improved
- [ ] Potential regression

Details:

- API call frequency
- Reconcile duration
- Memory/CPU impact

---

## Backward Compatibility

- [ ] Fully backward compatible
- [ ] Breaking change (describe below)

If breaking:

- Migration steps:
- Config changes required:

---

## Documentation

- [ ] README updated
- [ ] Docs updated
- [ ] Examples updated

---

## Checklist

- [ ] I have read the CONTRIBUTING guide
- [ ] Code follows project conventions
- [ ] `task ci` passes locally
- [ ] Tests added where needed
- [ ] CHANGELOG.md updated (if applicable)
- [ ] No sensitive data introduced
- [ ] I considered HA / leader election implications

---

## Reviewer Notes

Anything reviewers should focus on:

- Critical logic
- Risky areas
- Follow-ups
