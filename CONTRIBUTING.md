# Contributing to LoreLattice

## Before opening a change

- Use an issue or discussion for large product changes.
- Keep credentials, `.env`, database dumps and runtime snapshots out of Git.
- Preserve upstream license notices and update `UPSTREAM.md` when importing upstream code.

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
