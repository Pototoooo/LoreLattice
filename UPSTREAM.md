# Upstream provenance

## Source

- Upstream project: [Tencent/WeKnora](https://github.com/Tencent/WeKnora)
- Upstream default branch: `main`
- Local derivative name: `LoreLattice`
- Relationship: independent derivative; not an official Tencent distribution

LoreLattice keeps the inherited Git history so that individual lines, migrations and subsystems remain traceable to their original commits. Rebranding does not transfer authorship of the upstream platform.

## License handling

The root `LICENSE` is preserved from the upstream project. The main project is distributed under the MIT terms stated there, while listed third-party components retain their own license terms. Source headers and third-party notices must not be removed during future synchronization.

## Change policy

Changes specific to this repository should be recorded in `PROJECT_DIFF.md` and in ordinary Git commits. When importing a new upstream revision:

1. record the upstream commit or tag;
2. merge or cherry-pick without flattening the inherited history;
3. resolve product-specific differences explicitly;
4. rerun backend, frontend, migration and deployment checks;
5. update `PROJECT_DIFF.md` when the product boundary changes.
