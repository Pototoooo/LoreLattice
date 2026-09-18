# Contributing to LoreLattice

## Before opening a change

- Use an issue or discussion for large product changes.
- Keep credentials, `.env`, database dumps and runtime snapshots out of Git.
- Preserve applicable third-party license notices and document the source and scope when importing external code. This requirement does not imply that the whole project derives from any particular project.

## Local workflow

1. Create a focused branch.
2. Add or update tests with the implementation.
3. Run the checks relevant to the changed subsystem.
4. Explain the user impact, root cause and verification in the pull request.

Backend:

```bash
gofmt -w <changed-go-files>
go test ./path/to/changed/package
```

Frontend:

```bash
cd frontend
npm ci
npm run test
npm run type-check
npm run build
```

Deployment or proxy changes should additionally pass:

```bash
docker compose config
docker compose up -d
docker compose ps
curl http://localhost:8080/health
```

## Commit style

Use concise Conventional Commit subjects such as:

- `feat(billing): add usage aggregation`
- `fix(wiki): retry truncated model output`
- `docs: clarify five-model setup`

A pull request should state what changed, why it changed, how it was tested and any configuration or migration impact.
