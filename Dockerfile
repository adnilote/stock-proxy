
FROM golang:1.12.7-alpine3.9 as builder

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

CMD ["/runme"]  