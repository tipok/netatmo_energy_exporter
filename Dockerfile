FROM golang:1.21.5-alpine3.19 as builder

COPY ./  /data/

RUN cd /data && go build -o netatmo-exporter

FROM alpine:3.19

RUN addgroup -g 1001 -S appgroup && \
    adduser --u 1001 -S appuser appgroup

USER appuser
COPY --from=0 /data/netatmo-exporter /app/netatmo-exporter

ENTRYPOINT ["/app/netatmo-exporter"]
