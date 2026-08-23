FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY config.yaml ./
COPY bin/user-service-linux /usr/local/bin/user-service
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/user-service"]
