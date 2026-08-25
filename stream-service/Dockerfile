FROM alpine:3.20
RUN adduser -D -u 10001 app
RUN apk add --no-cache ca-certificates wget
USER app
WORKDIR /app
COPY config.yaml ./
COPY bin/stream /usr/local/bin/stream
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
  CMD wget -qO- http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/stream"]
CMD ["-config", "/app/config.yaml"]
