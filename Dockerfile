FROM gliderlabs/alpine

RUN apk-install bash go bzr git clang
RUN apk add --no-cache musl-dev
RUN  go get github.com/go-telegram-bot-api/telegram-bot-api github.com/tatsushid/go-fastping github.com/valyala/fasthttp
RUN mkdir -p /go/bin && chmod -R 777 /go

ADD ServerPingBot/ /go/ServerPingBot/

ENV GOPATH=/go/ServerPingBot/:/go:/root/go/
ENV PATH /go/bin:$PATH

WORKDIR /go/ServerPingBot

RUN go run main.go