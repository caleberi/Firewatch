FROM alpine:3.18

RUN apk add --no-cache curl libc6-compat unzip

RUN curl -fSL -o /tmp/loki.zip https://github.com/grafana/loki/releases/download/v3.5.3/loki-linux-amd64.zip \
    && unzip /tmp/loki.zip -d /usr/bin/ \
    && mv /usr/bin/loki-linux-amd64 /usr/bin/loki \
    && chmod a+x /usr/bin/loki \
    && rm /tmp/loki.zip

COPY ./loki.yml /etc/loki/loki.yml

RUN mkdir -p /loki

EXPOSE 3100

CMD ["/usr/bin/loki", "--config.file=/etc/loki/loki.yml"]