FROM golang:1.20 as build-binary
COPY . /project
WORKDIR /project
RUN go mod vendor \
  && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GO111MODULE=on go build \
    -v \
    -mod vendor \
    -o /project/bin/client /project

FROM alpine:3.18 as image
COPY --from=build-binary /project/bin/client /usr/local/bin/client
RUN apk --no-cache add ca-certificates \
    && chmod +x /usr/local/bin/client
ENTRYPOINT ["/usr/local/bin/client"]