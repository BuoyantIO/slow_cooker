FROM library/golang:1.7.3

WORKDIR /go/src/slow_cooker

ADD . /go/src/github.com/buoyantio/slow_cooker

RUN go build -o /go/bin/slow_cooker /go/src/github.com/buoyantio/slow_cooker/main.go

ENTRYPOINT ["/go/bin/slow_cooker"]
