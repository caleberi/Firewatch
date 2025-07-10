FROM alpine:3.22 AS builder
RUN apk add curl python3 py3-pip python3-dev build-base

ARG DEV_MODE="on"
# Metadata labels (OCI-compliant)
ARG BUILD_VERSION=unknown
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS

ARG PROM_LOGLEVEL=info
ARG PROM_PORT=9090
ARG PROM_RETENTION_TIME="30s"
ARG PROM_PID="prometheus"
ENV PROM_EXECUTABLE_VERSION_URL="https://github.com/prometheus/prometheus/releases/download/v3.4.1/prometheus-3.4.1.linux-amd64.tar.gz"


ARG GF_LOG_MODE
ARG GF_PLUGINS_PREINSTALL
ARG GF_GID="grafana"
ARG GRAFANA_PORT=3000
ARG GRAFANA_VERSION=v12.0.2
ARG GF_PATHS_CONFIG="/etc/grafana/grafana.ini"
ARG GF_PATHS_DATA="/var/lib/grafana"
ARG GF_PATHS_HOME="/usr/share/grafana"
ARG GF_PATHS_LOGS="/var/log/grafana"
ARG GF_PATHS_PLUGINS="/var/lib/grafana/plugins"
ARG GF_PATHS_PROVISIONING="/etc/grafana/provisioning"
ARG GF_INSTALL_IMAGE_RENDERER_PLUGIN="false"
ARG GF_INSTALL_PLUGINS="grafana-clock-panel 1.0.1,grafana-simple-json-datasource 1.3.5"
ARG GF_INSTALL_IMAGE_RENDERER_PLUGIN="false"
ENV GF_PATHS_PLUGINS="/var/lib/grafana-plugins"
ENV GF_PLUGIN_RENDERING_CHROME_BIN="/usr/bin/chrome"
ENV GF_PLUGIN_IMAGE_RENDER_URL="https://github.com/grafana/grafana-image-renderer/releases/latest/download/plugin-alpine-x64-no-chromium.zip"
ENV GF_EXECUTABLE_VERSION_URL="https://dl.grafana.com/oss/release/grafana-12.0.2.linux-amd64.tar.gz"

LABEL org.opencontainers.image.title="Observability Service"
LABEL org.opencontainers.image.description="Observability service for Configured applications"
LABEL org.opencontainers.image.version=${BUILD_VERSION}
LABEL org.opencontainers.image.vendor="Firewatch"
LABEL org.opencontainers.image.source="https://github.com/caleberi/fire-watch"
LABEL com.firewatch.service="service-watcher"

# install grafana for dashboard generation
ADD ${GF_EXECUTABLE_VERSION_URL} /tmp/
# install prometheus for metric generation 
ADD https://github.com/prometheus/prometheus/releases/download/v3.4.1/prometheus-3.4.1.linux-amd64.tar.gz /tmp/


RUN addgroup -S ${GF_GID} && \
    adduser -S -D -H -s /sbin/nologin -G ${GF_GID} ${GF_GID}

RUN if [$DEV_MODE = "on"]; then \
    tar -zxvf /tmp/grafana-12.0.2.linux-amd64.tar.gz \
    && mkdir -p /usr/share/grafana /etc/grafana \
    && mv /grafana-v12.0.2/bin/* /usr/share/grafana \
    && mv /grafana-v12.0.2/public /usr/share/grafana \
    && mv /grafana-v12.0.2/conf /etc/grafana \
    && chown -R root:root /usr/share/grafana \
    && chown -R root:root /etc/grafana \
    && chmod -R 755 /usr/share/grafana \
    && chmod -R 644 /etc/grafana \
    && chmod 755 /usr/share/grafana/grafana-server /usr/share/grafana/grafana-cli \
    && chown -R ${GF_GID}:${GF_GID} /usr/share/grafana /etc/grafana \
    fi

RUN if [$DEV_MODE = "on"]; then\
    mkdir -p "$GF_PATHS_PLUGINS" \
    && chown -R ${GF_GID}:${GF_GID} "$GF_PATHS_PLUGINS" \
    && if [ $GF_INSTALL_IMAGE_RENDERER_PLUGIN = "true"]; then \
    if grep -i -q alpine /etc/issue; then \
    apk add --no-cache udev ttf-opensans chromium && \
    ln -s /usr/bin/chromium-browser "$GF_PLUGIN_RENDERING_CHROME_BIN"; \
    else \
    echo "Cannot install image render plugin on non-alpine OS" \
    fi \
    fi\
    fi

USER ${GF_GID}

# Install dependencies for image renderer (only if enabled)
RUN if [ "$GF_INSTALL_IMAGE_RENDERER_PLUGIN" = "true" && $DEV_MODE = "on"]; then \
    # apk add --no-cache udev ttf-opensans chromium && \
    # ln -s /usr/bin/chromium-browser "$GF_PLUGIN_RENDERING_CHROME_BIN" && \
    /usr/share/grafana/grafana-cli --pluginsDir "$GF_PATHS_PLUGINS" \
    --pluginUrl ${GF_PLUGIN_IMAGE_RENDER_URL} \
    plugins install grafana-image-renderer; \
    else \
    echo "Skipping image renderer plugin installation"; \
    fi

RUN if [$DEV_MODE = "on"]; then\
    cp ./grafana.ini /etc/grafana/grafana.ini \
    echo "Building Grafana server with version ${BUILD_VERSION} for ${TARGETOS}/${TARGETPLATFORM}"\
    fi


RUN addgroup -S ${PROM_PID} && \
    adduser -S -D -H -s /sbin/nologin -G ${PROM_PID} ${PROM_PID}

USER ${PROM_PID}

RUN tar -xzf /tmp/prometheus-3.4.1.linux-amd64.tar.gz \
    && mv /prometheus-3.4.1.linux-amd64/prometheus /usr/bin/prometheus \
    && mv /prometheus-3.4.1.linux-amd64/promtool /usr/bin/promtool

# # Set working directory
WORKDIR /prometheus 


# # Copy configuration files and start script to the correct location
COPY  ./prometheus.yml /etc/prometheus/prometheus.yml
COPY  ./web.yml /etc/prometheus/web.yml
COPY  ./rules /etc/prometheus/rules
COPY  ./alerts /etc/prometheus/alerts
COPY  ./scrapes /etc/prometheus/scrapes


COPY ./observability.sh /observability.sh
COPY ./prom-builder.py ./prom-builder.py 
COPY ./requirements.txt  ./requirements.txt  
COPY ./populate.json /etc/prometheus/populate.json

# # Ensure configuration and data directories exist with correct permissions
RUN echo "Building Prometheus server with version ${BUILD_VERSION} for ${TARGETOS}/${TARGETPLATFORM}"


RUN python3 -m venv /venv \
    && /venv/bin/pip install --upgrade pip \
    && /venv/bin/pip install -r requirements.txt  

# # Make start script executable
RUN chmod +x /observability.sh

# # Expose Prometheus port (default 9090)
EXPOSE ${PROM_PORT}
EXPOSE ${GRAFANA_PORT}

# # Define volumes for persistent data and configuration
VOLUME ["/prometheus", "/etc/prometheus", "/var/lib/grafana"]

# # Healthcheck with adjusted timings
HEALTHCHECK --interval=15s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:${PROM_PORT}/-/healthy \
    && curl -f http://localhost:${GRAFANA_PORT}/api/health || exit 1

# # Entrypoint
ENTRYPOINT ["/observability.sh"]