FROM library/golang:1.7.1

WORKDIR /go/src/slow_cooker

ADD ./main.go /go/src/slow_cooker/
ADD ./vendor /go/src/slow_cooker/vendor

RUN go build -o /go/bin/slow_cooker /go/src/slow_cooker/main.go

ENTRYPOINT ["/go/bin/slow_cooker"]
