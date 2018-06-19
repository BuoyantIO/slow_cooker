FROM library/golang:1.10.3 as golang
WORKDIR /go/src/github.com/buoyantio/slow_cooker
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/slow_cooker .

FROM alpine:3.7
RUN apk --update upgrade && \
    apk add curl ca-certificates && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
ENV PATH=$PATH:/go/bin
COPY --from=golang /go/bin /go/bin
ENTRYPOINT ["slow_cooker"]
