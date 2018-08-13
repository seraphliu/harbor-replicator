FROM golang:latest AS builder

WORKDIR /go/src/github.com/seraphliu/harbor-replicator

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep
ADD . .
RUN dep ensure -vendor-only

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .


FROM docker:edge

COPY --from=builder /go/src/github.com/seraphliu/harbor-replicator/app /go/bin/app

ENTRYPOINT ["/go/bin/app"]
CMD ["--help"]
