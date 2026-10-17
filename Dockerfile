FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o preview-operator ./cmd/operator

FROM alpine:3.24

WORKDIR /app
COPY --from=builder /app/preview-operator /app/preview-operator

ENV PORT=8080
EXPOSE 8080

CMD ["/app/preview-operator"]
