FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build-for-linux-amd64

FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source="https://github.com/cnosuke/mcp-javascript-executor"
LABEL org.opencontainers.image.description="Sandboxed JavaScript executor MCP server"

WORKDIR /app

COPY --from=builder /app/config.yml /app/config.yml
COPY --from=builder /app/bin/mcp-javascript-executor-linux-amd64 /app/mcp-javascript-executor

USER nonroot:nonroot

ENTRYPOINT ["/app/mcp-javascript-executor"]

CMD ["stdioserver", "--config", "config.yml"]
