# Build the binary
FROM golang:1.13 as builder

WORKDIR /
COPY . .
RUN make unit-test
RUN make build

FROM oraclelinux:7-slim

RUN yum update -y && \
    yum install -y curl && \
    yum clean all

COPY --from=builder /go/bin/prometheus-pusher /prometheus-pusher
COPY LICENSE README.md THIRD_PARTY_LICENSES.txt /license/

ADD run.sh /

RUN chown -R 1000:1000 /prometheus-pusher /run.sh

USER 1000

ENTRYPOINT ["/bin/sh", "/run.sh"]
