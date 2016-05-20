FROM library/golang:1.6.0

WORKDIR /go/src/slow_cooker

ADD ./main.go /go/src/slow_cooker/
ADD ./Godeps/Godeps.json /go/src/slow_cooker/Godeps/

RUN go get github.com/tools/godep
RUN godep restore

RUN go build -o /go/bin/slow_cooker /go/src/slow_cooker/main.go

ENTRYPOINT ["/go/bin/slow_cooker"]
