# LoreLattice project differences

This document separates the current repository's focused work from the inherited platform. It is an evidence index, not a claim that the complete upstream system was authored here.

## Product direction

LoreLattice concentrates on three connected outcomes:

1. make the five model roles understandable and independently configurable;
2. record model use once while avoiding duplicate charges for BYOK/local models;
3. make RAG, Agent and Wiki behavior observable enough to diagnose from the UI and persisted data.

## Repository-specific work

### Unified AI Credits and BYOK accounting

- Introduced explicit billing modes: `platform`, `included`, `byok`, `local`.
- Added reservation, completion/release and persistent outbox semantics around Chat, Embedding, Rerank, VLM and ASR calls.
- Added a pricing catalog, result caching for non-chat model calls, usage aggregation by AI job and advanced raw-call details.
- BYOK/local calls remain visible in usage analytics but are not chargeable.
- Relevant commits: `96475c82`, `4a04dc1f`.

### Billing resilience

- The settings page loads the account overview before optional usage/invoice sections.
- A MeterForge outage no longer removes the entire billing page; read-only balance falls back to the local ledger.
- Platform-funded calls remain fail-closed when remote credit authorization is unavailable.

### Wiki generation reliability

- Wiki generation supplies an explicit completion budget for reasoning models.
- Empty responses and `finish_reason=length` are treated as incomplete and retried.
- OpenAI-compatible `reasoning_content` is retained for diagnosis instead of being discarded.

### Runtime resilience and UI correctness

- Nginx resolves the backend through Docker DNS instead of retaining a stale container IP after an app recreation.
- Empty image-viewer instances are no longer mounted in text-only conversations.
- Locale messages are compiled in tests to catch Vue i18n syntax failures.
- Placeholder Langfuse credentials stay disabled until real keys are configured.

## Verification evidence

The latest local regression covered:

- 221 frontend tests, Vue type checking and a Node.js 24 production build;
- focused Go tests for billing, Wiki ingest and OpenAI-compatible chat parsing;
- Docker health for app, DocReader, PostgreSQL, Redis, MeterForge and ClickHouse;
- browser flows for model settings, agents, knowledge-base upload, RAG citations, global search, Wiki pages/graph and billing usage;
- database cleanup of temporary QA accounts, sessions, knowledge bases, vectors and credentials.

Some repository-wide integration tests require external services or test-only SSRF allowlists. Their environment dependencies should not be represented as passing when those services are absent.

## Attribution boundary

The general RAG platform, document pipeline, Agent runtime, connectors, UI foundation and most repository history originate from the upstream project. LoreLattice-specific claims should be limited to changes supported by commits, diffs, tests and deployment evidence in this repository.
