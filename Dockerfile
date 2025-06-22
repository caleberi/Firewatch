FROM alpine:3.22 AS builder
RUN apk add curl python3 py3-pip python3-dev build-base

# Metadata labels (OCI-compliant)
ARG BUILD_VERSION=unknown
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG PROM_LOGLEVEL=info
ARG PROM_PORT=9090
ARG PROM_RETENTION_TIME="30s"
LABEL org.opencontainers.image.title="Cudium Prometheus Service"
LABEL org.opencontainers.image.description="Prometheus service for Cudium applications"
LABEL org.opencontainers.image.version=${BUILD_VERSION}
LABEL org.opencontainers.image.vendor="Cudium"
LABEL org.opencontainers.image.source="https://github.com/cudium/prometheus-service"
LABEL com.cudium.service="service-watcher"



RUN curl -LO https://github.com/prometheus/prometheus/releases/download/v3.4.1/prometheus-3.4.1.linux-amd64.tar.gz \
    && tar -xzf prometheus-3.4.1.linux-amd64.tar.gz \
    && mv prometheus-3.4.1.linux-amd64/prometheus /usr/bin/prometheus \
    && mv prometheus-3.4.1.linux-amd64/promtool /usr/bin/promtool \
    && rm -rf prometheus-3.4.1.linux-amd64.tar.gz prometheus-3.4.1.linux-amd64

# # Set working directory
WORKDIR /prometheus

# # Ensure configuration and data directories exist with correct permissions
RUN echo "Building Prometheus server with version ${BUILD_VERSION} for ${TARGETOS}/${TARGETPLATFORM}"

# # Copy configuration files and start script to the correct location
COPY  ./prometheus.yml /etc/prometheus/prometheus.yml
COPY  ./web.yml /etc/prometheus/web.yml
COPY  ./rules /etc/prometheus/rules
COPY  ./alerts /etc/prometheus/alerts
COPY  ./scrapes /etc/prometheus/scrapes


COPY  ./run-prometheus.sh /run-prometheus.sh
COPY ./prom-builder.py ./prom-builder.py 
COPY ./requirements.txt  ./requirements.txt  
COPY ./populate.json /etc/prometheus/populate.json

RUN python3 -m venv /venv \
    && /venv/bin/pip install --upgrade pip \
    && /venv/bin/pip install -r requirements.txt  

# # Make start script executable
RUN chmod +x /run-prometheus.sh

# # Expose Prometheus port (default 9090)
EXPOSE ${PROM_PORT}

# # Define volumes for persistent data and configuration
VOLUME ["/prometheus", "/etc/prometheus"]

# # Healthcheck with adjusted timings
HEALTHCHECK --interval=15s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:${PROM_PORT}/-/healthy || exit 1

# # Entrypoint
ENTRYPOINT ["/run-prometheus.sh"]