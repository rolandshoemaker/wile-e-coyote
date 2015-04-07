FROM golang:1.4.2

MAINTAINER R.B. Shoemaker "rolandshoemaker@gmail.com"

RUN mkdir -p /go/src/github.com/rolandshoemaker/wile-e-coyote/chall-srv
COPY . /go/src/github.com/rolandshoemaker/wile-e-coyote/chall-srv

RUN go install github.com/rolandshoemaker/wile-e-coyote/chall-srv

RUN dnsmasq -k -R --address=/#/127.0.0.1

CMD ["/go/bin/chall-srv"]
