FROM golang:1.10 as builder

WORKDIR ${GOPATH}/src/github.com/dispatchframework/dispatch-events-vcenter

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /dispatch-events-vcenter


FROM scratch

ADD cacert-2018-03-07.pem /etc/ssl/certs/
COPY --from=builder /dispatch-events-vcenter /

ENTRYPOINT [ "/dispatch-events-vcenter" ]
