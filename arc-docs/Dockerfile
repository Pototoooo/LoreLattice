# Build from repository root:
#   docker build --platform linux/amd64 -f arc-docs/Dockerfile -t pototoooo/lorelattice-docs:latest .
#
# Gateway (preserve /docs/ prefix):
#   location /docs/ { proxy_pass http://lorelattice-docs:80/docs/; }
#
# VitePress reads VERSION from the repo root (see .vitepress/version.ts), so the
# builder stage mirrors the monorepo layout instead of flattening arc-docs/.
FROM node:22-bookworm-slim AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY VERSION VERSION
COPY arc-docs/package.json arc-docs/package-lock.json arc-docs/
RUN cd arc-docs && npm ci

COPY arc-docs/ arc-docs/
RUN cd arc-docs && npm run build

# Align with internal frontend image: stable-alpine + custom entrypoint + bash.
FROM nginx:stable-alpine AS production-stage

COPY --from=builder /build/arc-docs/.vitepress/dist /usr/share/nginx/html

RUN apk add --no-cache bash

COPY arc-docs/nginx.conf /etc/nginx/conf.d/default.conf
COPY arc-docs/docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/docker-entrypoint.sh"]
