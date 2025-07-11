
# Metadata labels (OCI-compliant)
ARG BASE_OS_IMAGE=alpine
ARG BASE_OS_VERSION=3.22


FROM ${BASE_OS_IMAGE}:${BASE_OS_VERSION} AS builder
RUN apk add curl python3 py3-pip python3-dev build-base


ARG BUILD_VERSION=unknown
ARG BUILD_PLATFORM
ARG TARGET_OS
ARG TARGET_PLATFORM
ARG DEV_MODE
ARG RM_TEMP

ARG PROM_LOGLEVEL=info
ARG PROM_PORT=9090
ARG PROM_RETENTION_TIME="30s"
ARG PROM_PID="prometheus"


ARG GF_PATHS_DATA
ARG GF_PATHS_HOME
ARG GF_PATHS_LOGS
ARG GF_PATHS_PLUGINS

ARG GF_PLUGINS_PREINSTALL
ARG GF_GID="grafana"
ARG GF_LOG_MODE="console"
ARG GRAFANA_PORT=3000
ARG GRAFANA_VERSION=v12.0.2
ARG GF_PATHS_CONFIG="/etc/grafana"
ARG GF_PATHS_PROVISIONING="/etc/grafana/provisioning"
ARG GF_INSTALL_IMAGE_RENDERER_PLUGIN="false"
ARG GF_INSTALL_PLUGINS="grafana-clock-panel 1.0.1,grafana-simple-json-datasource 1.3.5"
ARG GF_INSTALL_IMAGE_RENDERER_PLUGIN="false"

ENV GF_PATHS_PLUGINS="/var/lib/grafana-plugins"
ENV GF_PLUGIN_RENDERING_CHROME_BIN="/usr/bin/chrome"

ENV GF_PLUGIN_IMAGE_RENDER_URL="https://github.com/grafana/grafana-image-renderer/releases/latest/download/plugin-alpine-x64-no-chromium.zip"
ENV GF_EXECUTABLE_VERSION_URL="https://dl.grafana.com/oss/release/grafana-12.0.2.linux-amd64.tar.gz"
ENV PROM_EXECUTABLE_VERSION_URL="https://github.com/prometheus/prometheus/releases/download/v3.4.1/prometheus-3.4.1.linux-amd64.tar.gz"


LABEL org.opencontainers.image.title="Observability Service"
LABEL org.opencontainers.image.description="Observability service for Configured applications"
LABEL org.opencontainers.image.version=$BUILD_VERSION
LABEL org.opencontainers.image.vendor="Firewatch"
LABEL org.opencontainers.image.source="https://github.com/caleberi/fire-watch"
LABEL com.firewatch.service="service-watcher"

# install grafana for dashboard generation
ADD ${GF_EXECUTABLE_VERSION_URL} /tmp/
# install prometheus for metric generation 
ADD ${PROM_EXECUTABLE_VERSION_URL} /tmp/


RUN addgroup -S ${GF_GID} && \
    adduser -S -D -H -s /sbin/nologin -G ${GF_GID} ${GF_GID}

# Install Grafana conditionally based on DEV_MODE
RUN if [ "${DEV_MODE}" = "on" ]; then \
    [ -f /tmp/grafana-12.0.2.linux-amd64.tar.gz ] || { echo "Grafana tarball not found"; exit 1; } && \
    tar -zxvf /tmp/grafana-12.0.2.linux-amd64.tar.gz || { echo "Failed to extract Grafana tarball"; exit 1; } && \
    GRAFANA_DIR=$(tar -tf /tmp/grafana-12.0.2.linux-amd64.tar.gz | head -n1 | cut -d'/' -f1) && \
    mkdir -p /usr/share/grafana /etc/grafana /grafana && \
    mv "/${GRAFANA_DIR}/bin"/* /usr/share/grafana && \
    mv "/${GRAFANA_DIR}/public" /usr/share/grafana && \
    mv "/${GRAFANA_DIR}/conf" /usr/share/grafana && \
    chown -R ${GF_GID}:${GF_GID} /usr/share/grafana /etc/grafana /grafana && \
    chmod -R 755 /usr/share/grafana /grafana && \
    chmod -R 644 /etc/grafana && \
    chmod 755 /usr/share/grafana/grafana-server /usr/share/grafana/grafana-cli /usr/share/grafana/grafana && \
    chown -R "${GF_GID}:${GF_GID}" /usr/share/grafana /etc/grafana && \
    rm -rf "/${GRAFANA_DIR}"; \
    fi

# Set up Grafana plugins directory and install image renderer plugin if enabled
RUN if [ "$DEV_MODE" = "on" ]; then \
    mkdir -p "$GF_PATHS_PLUGINS" && \
    chown -R ${GF_GID}:${GF_GID} "$GF_PATHS_PLUGINS" && \
    if [ "$GF_INSTALL_IMAGE_RENDERER_PLUGIN" = "true" ]; then \
    if grep -i -q alpine /etc/issue; then \
    apk add --no-cache udev ttf-opensans chromium && \
    ln -s /usr/bin/chromium-browser "$GF_PLUGIN_RENDERING_CHROME_BIN" && \
    su -s /bin/sh ${GF_GID} -c "/usr/share/grafana/grafana cli --pluginsDir \"$GF_PATHS_PLUGINS\" --pluginUrl \"$GF_PLUGIN_IMAGE_RENDER_URL\" plugins install grafana-image-renderer"; \
    else \
    echo "Cannot install image render plugin on non-alpine OS"; \
    fi; \
    else \
    echo "Skipping image renderer plugin installation"; \
    fi; \
    fi

COPY --chown=${GF_GID}:${GF_GID} grafana.ini ${GF_PATHS_CONFIG}/grafana.ini
RUN chmod 755 /etc/grafana && chmod 644 /etc/grafana/grafana.ini

USER root

RUN addgroup -S ${PROM_PID} && \
    adduser -S -D -H -s /sbin/nologin -G ${PROM_PID} ${PROM_PID}

RUN tar -xzf /tmp/prometheus-3.4.1.linux-amd64.tar.gz \
    && mv /prometheus-3.4.1.linux-amd64/prometheus /usr/bin/prometheus \
    && mv /prometheus-3.4.1.linux-amd64/promtool /usr/bin/promtool \
    && rm -rf /prometheus-3.4.1.linux-amd64

USER ${PROM_PID}

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

USER root

RUN python3 -m venv /venv \
    && /venv/bin/pip install --upgrade pip \
    && /venv/bin/pip install -r requirements.txt  

# # Make start script executable
RUN chmod +x /observability.sh

RUN if [ ! -z "${DEV_MODE}" ]; then \
    rm -rf /tmp; \
    fi

# # Expose Prometheus port (default 9090)
EXPOSE ${PROM_PORT}
EXPOSE ${GRAFANA_PORT}

# # Define volumes for persistent data and configuration
VOLUME ["/prometheus", "/etc/prometheus", "/var/lib/grafana"]

# # Healthcheck with adjusted timings
HEALTHCHECK --interval=15s --timeout=10s --start-period=30s --retries=3 \
    CMD if [ "$DEV_MODE" = "on" ]; then \
    curl -f http://localhost:${PROM_PORT}/-/healthy && curl -f http://localhost:${GRAFANA_PORT}/api/health; \
    else \
    curl -f http://localhost:${PROM_PORT}/-/healthy; \
    fi || exit 1


ENTRYPOINT ["sh", "-c", "/observability.sh --DEV_MODE \"$DEV_MODE\" --GF_LOG_MODE \"$GF_LOG_MODE\" --GF_PATHS_PLUGINS \"$GF_PATHS_PLUGINS\" --GF_PATHS_DATA \"$GF_PATHS_DATA\" --GF_PATHS_LOGS \"$GF_PATHS_LOGS\" --GF_PATHS_PROVISIONING \"$GF_PATHS_PROVISIONING\""]