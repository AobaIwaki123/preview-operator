FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o sample-app .

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /app/sample-app /app/sample-app

ENV PORT=8080
EXPOSE 8080

CMD ["/app/sample-app"]
