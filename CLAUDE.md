# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build              # Compile manager binary to bin/manager
make test               # Run tests with coverage (generates cover.out)
make run                # Run controller locally against current kubeconfig
make manifests          # Regenerate CRD and RBAC manifests (run after changing api/v1/*_types.go)
make generate           # Regenerate DeepCopy methods (run after changing api/v1/*_types.go)
make fmt                # gofmt
make vet                # go vet
make docker-build       # Build container image (runs tests first)
make install            # Install CRDs into the current cluster
make deploy             # Deploy controller to the current cluster
```

Run a single test:
```bash
go test ./template_io/ -run TestTemplateParsing -v
```

Tests use ginkgo v2 + gomega. The controllers test suite uses envtest (local API server, no real cluster needed).

## Architecture

This is a Kubernetes operator (controller-runtime/Kubebuilder) managing three CRDs in API group `helx.renci.org/v1`:

- **HelxApp** — application template (images, ports, env, volumes, security context)
- **HelxInst** — per-user instance request referencing an app + user; triggers workload creation. Has its own `environment` map — highest precedence in the three-way merge (app < user < inst).
- **HelxUser** — user record; `userHandle` URL fetches security context (uid/gid) via HTTP. Has `environment` and `volumes` fields merged with app/inst (app < user < inst precedence).

The three CRDs arrive independently and in any order. The controller maintains an **in-memory relational graph** (`helxapp_operations` package) with bidirectional associations. Workload objects (Deployment, PVCs, Services) are only created when a complete triple (app + user + instance) exists. If a resource arrives late, the reconciler for the arriving resource picks up waiting instances and completes them.

### Key packages

| Package | Role |
|---------|------|
| `api/v1/` | CRD type definitions (spec, status, DeepCopy) |
| `controllers/` | Three reconcilers, one per CRD kind |
| `helxapp_operations/` | In-memory object graph, artifact generation, cluster CRUD |
| `template_io/` | Template type definitions, rendering utilities, volume DSL parsing |
| `templates/` | Go templates (deployment, pod, container, pvc, service) |
| `connect/` | HTTP client for fetching user info from `userHandle` URLs |
| `plugin/ldap/` | Separate Go module — LDAP HTTP service resolving user security context |

### Artifact generation pipeline

1. Resolve app + user from graph (both must exist)
2. `transformApp()` converts CRD services → `template_io.Container` structs
3. Build `template_io.System` context with env, security context, volumes
4. Render templates — **double-pass**: Go template engine renders YAML, then re-renders the result as a template (allows `{{ .system.UserName }}` in field values)
5. Decode YAML → typed K8s objects → `CreateOrUpdateResource` (create or JSON-patch)

### Volume DSL

Volumes use a mini-language: `[scheme://]src:mountPath[#subPath][,option=value...]`
- Schemes: `pvc` (default), `nfs`
- Options: `retain`, `rwx`/`rox`/`rwop`, `size`, `storageClass`

### Security context priority

1. `HelxInst.Spec.SecurityContext` (explicit override)
2. HTTP GET to `HelxUser.Spec.UserHandle` URL
3. Omitted

## Important Patterns

- After modifying `api/v1/*_types.go`, always run `make manifests generate`
- Templates are parsed at startup via `Initialize()` in main.go from the `/templates` directory
- Objects with label `helx.renci.org/retain: "true"` survive instance deletion
- All derived objects share label `helx.renci.org/id: <UUID>` for set-based lookup/deletion
- PVC patches filter out `remove` operations to protect bound claims
