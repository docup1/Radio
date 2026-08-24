FROM golang:1.22-alpine AS build
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/sender ./cmd/sender

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/sender /usr/local/bin/sender
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
  CMD wget -qO- http://localhost:8081/ || exit 1
ENTRYPOINT ["/usr/local/bin/sender"]
CMD ["-config", "/app/config.yaml"]
