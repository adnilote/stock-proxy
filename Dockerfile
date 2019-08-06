
FROM golang:1-alpine3.10 as builder

RUN mkdir /proxy
ADD . /proxy
WORKDIR /proxy

RUN apk update && apk add git
RUN go get -d -v ./...

RUN apk add --update gcc musl-dev
RUN go test -c ./ -o /out/tests/stock-proxy.test
RUN go build -o /out/runme

FROM alpine:3.9 as release
COPY --from=builder /out/ .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD ["/runme"]  