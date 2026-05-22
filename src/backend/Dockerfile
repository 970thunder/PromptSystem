FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY sql ./sql

RUN CGO_ENABLED=0 GOOS=linux go build -o /promptos-api ./cmd/api

FROM alpine:3.20

RUN adduser -D -H promptos

WORKDIR /app
RUN mkdir -p /app/uploads && chown -R promptos:promptos /app
COPY --from=builder /promptos-api /usr/local/bin/promptos-api
COPY --from=builder /app/sql /app/sql

USER promptos

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/promptos-api"]
