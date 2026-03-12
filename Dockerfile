FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o controller ./cmd/controller

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /
COPY --from=builder /app/controller .

USER 65532:65532

ENTRYPOINT ["/controller"]
