FROM golang:1.25-alpine AS builder

WORKDIR /app

# 为什么需要 vendor：go.mod 用 replace 指向仓库外的 ../isoumao-shared（平台统一契约），
# 该目录不在构建上下文里，纯 COPY go.mod 构建必然失败（replacement directory does not exist）。
# 因此发布前必须先执行 `go mod vendor`，把契约源码纳入构建上下文，再用 -mod=vendor 构建。
# vendor/ 是生成物、不入库：入库会让契约出现第二份副本，正是要避免的事。
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
COPY sql ./sql
COPY vendor ./vendor

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /promptos-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /promptos-integrity-audit ./cmd/integrity-audit
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /promptos-maintenance ./cmd/maintenance

FROM alpine:3.20

RUN adduser -D -H promptos

WORKDIR /app
RUN mkdir -p /app/uploads && chown -R promptos:promptos /app
COPY --from=builder /promptos-api /usr/local/bin/promptos-api
COPY --from=builder /promptos-integrity-audit /usr/local/bin/promptos-integrity-audit
COPY --from=builder /promptos-maintenance /usr/local/bin/promptos-maintenance
COPY --from=builder /app/sql /app/sql

USER promptos

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/promptos-api"]
