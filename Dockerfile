# syntax=docker/dockerfile:1

# ---- build: compila um binário estático (sem cgo) ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# Copiar go.mod/go.sum antes do código faz o download das dependências virar
# uma camada cacheada: só refaz quando as dependências mudam.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/api ./cmd/api

# ---- runtime: distroless static (~2 MB), sem shell, sem gerenciador de pacotes ----
# Já inclui certificados CA, tzdata e usuário nonroot (uid 65532).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
# Sem curl/wget na imagem: o próprio binário faz o healthcheck.
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
