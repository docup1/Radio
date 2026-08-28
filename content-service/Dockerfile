FROM alpine:3.20
RUN apk add --no-cache ffmpeg
RUN adduser -D -u 10001 app
RUN mkdir -p /data/chunks /data/media && chown -R app:app /data
USER app
WORKDIR /app
COPY config.yaml ./
COPY docs/ /app/docs/
COPY bin/content-service /usr/local/bin/content-service
EXPOSE 8080 8081 8082
ENTRYPOINT ["content-service"]
