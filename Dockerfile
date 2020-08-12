ARG BUILDPLATFORM=linux/amd64

FROM --platform=$BUILDPLATFORM golang:1.14.2-stretch as build
WORKDIR /slow_cooker

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -installsuffix cgo -o ./slow_cooker

FROM alpine:3.12
RUN apk --update upgrade && \
    apk add ca-certificates curl nghttp2 && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*
COPY --from=build /slow_cooker/slow_cooker /slow_cooker/
ENTRYPOINT ["/slow_cooker/slow_cooker"]
