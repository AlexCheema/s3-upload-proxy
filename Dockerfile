FROM golang:1.14.6-alpine3.12@sha256:70d49538b8f7acd5b71e84f81bebf7667f25017308324a91f312a9830a618f3d AS build

ENV  CGO_ENABLED 0
WORKDIR /code
ADD  . ./
RUN  go install

FROM alpine:3.11.6
RUN apk add --no-cache ca-certificates mailcap
COPY --from=build /go/bin/s3-upload-proxy /usr/bin/s3-upload-proxy
ENTRYPOINT ["/usr/bin/s3-upload-proxy"]
