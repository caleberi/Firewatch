FROM alpine:3.18

RUN apk add --no-cache curl libc6-compat tar unzip

RUN curl -fSL -o /tmp/promtail.zip https://github.com/grafana/loki/releases/download/v3.5.3/promtail-linux-arm64.zip \
    && unzip /tmp/promtail.zip -d /usr/bin/ \
    && mv /usr/bin/promtail-linux-arm64 /usr/bin/promtail \
    && chmod a+x /usr/bin/promtail \
    && rm /tmp/promtail.zip

COPY ./promtail.yml /etc/promtail/promtail.yml

RUN mkdir -p /promtail

EXPOSE 9080

CMD ["/usr/bin/promtail", "--config.file=/etc/promtail/promtail.yml"]