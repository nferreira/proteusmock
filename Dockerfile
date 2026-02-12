# ── Stage 0: dashboard — build React SPA ────────────────────────────
FROM node:22-bookworm-slim AS dashboard

WORKDIR /dashboard
COPY ui/dashboard/package.json ui/dashboard/package-lock.json ./
RUN npm ci
COPY ui/dashboard/ .
RUN npm run build

# ── Stage 1: deps — cache Go modules ────────────────────────────────
FROM golang:1.25-bookworm AS deps

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ── Stage 2: build — test + compile ─────────────────────────────────
FROM deps AS build

COPY . .
COPY --from=dashboard /dashboard/dist ui/dashboard/dist/

ARG RUN_TESTS=true
ARG TEST_TAGS=""

RUN if [ "$RUN_TESTS" = "true" ]; then \
      if [ -n "$TEST_TAGS" ]; then \
        go test -tags="$TEST_TAGS" -race -count=1 ./...; \
      else \
        go test -race -count=1 ./...; \
      fi; \
    fi

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /bin/proteusmock ./cmd/proteusmock
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o /bin/healthcheck ./cmd/healthcheck

# ── Stage 3: production — minimal runtime ───────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot AS production

COPY --from=build /bin/proteusmock /proteusmock
COPY --from=build /bin/healthcheck /healthcheck
COPY --from=build /src/mock /mock

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/healthcheck"]

ENTRYPOINT ["/proteusmock"]
CMD ["--root", "/mock", "--port", "8080", "--log-level", "info"]
