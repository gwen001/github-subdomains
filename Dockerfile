FROM golang:1.19.3-alpine AS build-env
MAINTAINER devopscoder331 "https://github.com/devopscoder331"

COPY ./ /go/app
WORKDIR /go/app
RUN go build -o /go/app/github-subdomains

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build-env /go/app/github-subdomains /usr/local/bin/github-subdomains
ENTRYPOINT ["github-subdomains"]