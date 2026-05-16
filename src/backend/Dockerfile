FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /promptos-api ./cmd/api

FROM alpine:3.20

RUN adduser -D -H promptos

WORKDIR /app
COPY --from=builder /promptos-api /usr/local/bin/promptos-api

USER promptos

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/promptos-api"]
