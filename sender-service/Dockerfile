FROM alpine:3.20
RUN adduser -D -u 10001 app
RUN apk add --no-cache ca-certificates wget
USER app
WORKDIR /app
COPY config.yaml ./
COPY bin/sender /usr/local/bin/sender
EXPOSE 8081
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
  CMD wget -qO- http://localhost:8081/ || exit 1
ENTRYPOINT ["/usr/local/bin/sender"]
CMD ["-config", "/app/config.yaml"]
