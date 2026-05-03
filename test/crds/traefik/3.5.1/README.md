# Traefik v3.5.1 CRDs (vendored)

This directory vendors a minimal set of Traefik CRDs for tests:

- traefik.io_ingressroutes.yaml
- traefik.io_middlewares.yaml
- traefik.io_traefikservices.yaml

Notes

- These CRDs are intentionally permissive (preserve unknown fields) to allow envtest to accept any `spec` used by the controller.
- For strict schema validation pinned to an official tag (v3.5.1), run `scripts/update-traefik-crds.sh` to replace these stubs with upstream CRDs byte-for-byte.
- Keep line endings as LF.
