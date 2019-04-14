FROM alpine:3.9

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/kanary /usr/local/bin/kanary
