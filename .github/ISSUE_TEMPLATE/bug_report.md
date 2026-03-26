---
name: Bug report
about: Report a reproducible problem in the Pangolin Kubernetes Controller
labels: [bug]
---

# Description

Provide a clear and concise description of the issue.

---

## Expected Behavior

What should have happened?

---

## Actual Behavior

What actually happened?

Include:

* Errors
* Unexpected state in Kubernetes
* Missing/incorrect Traefik resources

---

## Reproduction Steps

Minimal steps to reproduce:

1. Configure controller with:
   * `CONFIG_ENDPOINT`
   * Any relevant env vars
2. Deploy controller
3. Trigger event / wait for reconcile
4. Observe issue

---

## Observed Kubernetes State

Provide relevant resource outputs:

```bash
kubectl get ingressroutes,middlewares,traefikservices -A -o yaml
```

---

## Logs

Include **controller logs** with log level if possible:

```text
# logs here
```

If relevant, include:

* Leader election logs
* Reconciliation loop logs
* Apply / SSA conflicts
* GC actions

---

## Pangolin API Interaction

* Endpoint: `CONFIG_ENDPOINT`
* TLS/mTLS enabled: yes/no
* Response status: (200 / 304 / error)
* Any API errors?

---

## Environment

* Controller version: `vX.Y.Z`
* Kubernetes version: `vX.Y`
* Traefik CRD version: `v1alpha1`
* Installation method: (Helm / Kustomize)
* Cluster type: (Talos, k3s, EKS, AKS, etc.)

---

## Additional Context

Anything else that helps:

* Regression? (worked before?)
* Related issues
* Workarounds
