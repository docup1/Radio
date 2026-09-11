FROM golang:1.26-alpine

RUN apk add --no-cache ca-certificates git gcc libc-dev

WORKDIR /app

COPY user-service/     user-service/
COPY content-service/  content-service/
COPY stream-service/   stream-service/
COPY sender-service/   sender-service/
COPY gateway/          gateway/
COPY tests/run_all.sh  tests/run_all.sh
COPY tests/fixtures/   tests/fixtures/

ENV GOTOOLCHAIN=auto

ENTRYPOINT ["/app/tests/run_all.sh"]