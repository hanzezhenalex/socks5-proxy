FROM golang:1.19 as build

WORKDIR /usr/src/app

COPY . .

RUN make binaries

FROM golang:1.19 as proxyServer

COPY --from=build /go/bin/s5proxy /usr/bin/s5proxy

ENTRYPOINT ["/usr/bin/s5proxy"]