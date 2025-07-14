FROM accelbyte/golang-builder:1.24.1-alpine

ENV PATH="${GOPATH}/bin:${PATH}"

RUN apk add --no-cache yarn git bash

WORKDIR /app

ENTRYPOINT []