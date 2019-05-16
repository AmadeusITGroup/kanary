FROM golang:alpine as builder

ADD main.go .
RUN CGO_ENABLED=0 go build -i -installsuffix cgo -ldflags '-w' -o /simple-server .

FROM alpine:3.9
RUN apk upgrade --update --no-cache

USER nobody

COPY --from=builder /simple-server /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/simple-server" ]
