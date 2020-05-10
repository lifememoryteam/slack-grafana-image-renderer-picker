FROM golang:latest as builder
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /go/src/github.com/atpons/slack-grafana-image-renderer-picker
COPY . .
RUN go build  ./cmd/grasla/grasla.go

FROM frolvlad/alpine-glibc

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/atpons/slack-grafana-image-renderer-picker/grasla /usr/local/bin/grasla

ENTRYPOINT ["/usr/local/bin/grasla"]