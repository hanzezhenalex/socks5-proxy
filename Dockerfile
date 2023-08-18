FROM golang:1.19 as build

WORKDIR /usr/src/app

COPY . .

RUN make binaries

FROM golang:1.19 as proxyServer

COPY --from=build /go/bin/s5server /usr/bin/s5server

ENTRYPOINT ["/usr/bin/s5server"]

FROM golang:1.19 as proxyAgent

COPY --from=build /go/bin/s5agent /usr/bin/s5agent

ENTRYPOINT ["/usr/bin/s5agent"]