FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Regenerate the OpenAPI document from the handler annotations (JSON + YAML only,
# no embedded docs.go) so it never drifts from the code.
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.4 && \
    swag init -g api/router.go -o docs --parseInternal --ot json,yaml
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/gateway .

FROM alpine:3.20
RUN adduser -D -u 10002 app
USER app
WORKDIR /app
COPY --from=build /out/gateway /usr/local/bin/gateway
COPY config.yaml ./
COPY --from=build /src/docs /app/docs
# Built frontend (produced by `make prod` via `make frontend-dist`). In dev this
# directory is empty and STATIC_DIR is not set, so the gateway only proxies the API.
COPY dist /app/dist
ENV STATIC_DIR=/app/dist
EXPOSE 8080
CMD ["/usr/local/bin/gateway"]
