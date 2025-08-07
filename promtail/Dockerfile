FROM alpine:3.18


RUN apk add --no-cache \
    curl \
    libc6-compat


RUN curl -fSL -o /usr/bin/promtail https://github.com/grafana/loki/releases/latest/download/promtail-linux-amd64 \
    && chmod a+x /usr/bin/promtail


COPY promtail.yml /etc/promtail/promtail.yml


RUN mkdir -p /promtail

CMD ["/usr/bin/promtail", "--config.file=/etc/promtail/promtail.yml"]