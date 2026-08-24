FROM alpine:3.20
RUN adduser -D -u 10001 app
RUN mkdir -p /data/chunks /data/media && chown -R app:app /data
USER app
WORKDIR /app
COPY config.yaml ./
COPY docs/ /app/docs/
COPY bin/public /usr/local/bin/public
COPY bin/stream /usr/local/bin/stream
COPY bin/access /usr/local/bin/access
EXPOSE 8080 8081 8082
