FROM alpine:3.18


RUN apk add --no-cache \
    curl \
    libc6-compat


RUN curl -fSL -o /usr/bin/loki https://github.com/grafana/loki/releases/latest/download/loki-linux-amd64 \
    && chmod a+x /usr/bin/loki


COPY loki.yml /etc/loki/loki.yml

RUN mkdir -p /loki


EXPOSE 3100


CMD ["/usr/bin/loki", "--config.file=/etc/loki/loki.yml"]