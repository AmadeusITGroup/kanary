FROM alpine:3.9

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/simple-server /usr/local/bin/simple-server

ENTRYPOINT [ "/usr/local/bin/simple-server" ]
