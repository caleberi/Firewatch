# Define build arguments for base OS image and version
ARG BASE_OS_IMAGE=alpine
ARG BASE_OS_VERSION=3.22

# Stage 1: Build Python environment and extract Prometheus binaries
FROM ${BASE_OS_IMAGE}:${BASE_OS_VERSION} AS builder
# Install curl, Python 3, pip, and tar without caching packages
RUN apk add --no-cache curl python3 py3-pip tar
# Copy requirements.txt to the working directory
COPY requirements.txt /requirements.txt
# Create a virtual environment and install dependencies
RUN python3 -m venv /venv && \
    /venv/bin/pip install --no-cache-dir -r requirements.txt && \
    rm -rf /root/.cache/pip && \
    ls -l /venv/bin/python3
# Download and extract only the required Prometheus binaries
ARG PROM_EXECUTABLE_VERSION_URL="https://github.com/prometheus/prometheus/releases/download/v3.4.1/prometheus-3.4.1.linux-amd64.tar.gz"
ARG PROM_ALERTMANAGER_EXECUTABLE_URL="https://github.com/prometheus/alertmanager/releases/download/v0.28.1/alertmanager-0.28.1.linux-amd64.tar.gz"
RUN curl -sL ${PROM_EXECUTABLE_VERSION_URL} | tar -xzf - -C /usr/bin --strip-components=1 prometheus-3.4.1.linux-amd64/prometheus 
RUN curl -sL ${PROM_ALERTMANAGER_EXECUTABLE_URL} | tar -xzf - -C /usr/bin --strip-components=1 alertmanager-0.28.1.linux-amd64/alertmanager 

# Stage 2: Final image
FROM ${BASE_OS_IMAGE}:${BASE_OS_VERSION}
# Install curl (required for healthcheck)
RUN apk add --no-cache curl python3
# Add metadata labels for the image
LABEL maintainer="Firewatch Labs <hello@firewatch.com>"
LABEL org.opencontainers.image.title="Observability Service"
LABEL org.opencontainers.image.description="Observability service for Configured applications"
LABEL org.opencontainers.image.vendor="Firewatch"
LABEL org.opencontainers.image.source="https://github.com/caleberi/firewatch"
LABEL com.firewatch.service="service-watcher"
# Define Prometheus process ID and environment variables for configuration
ARG PROM_PID="prometheus"
ARG PROM_PORT=9090
# don't change the alertmanager port
ARG ALERT_MANAGER_PORT=9093
ARG PROM_SETUP_DIR=/etc/prometheus
ENV PROM_LOGLEVEL="info" \
    PROM_RETENTION_TIME="30s" \
    PROM_RETENTION_SIZE="512MB" \
    PROMETHEUS_CONFIG_FILE=$PROM_SETUP_DIR/prometheus.yml \
    PROMETHEUS_WEB_CONFIG_FILE=$PROM_SETUP_DIR/web.yml \
    POPULATION_FILE=$PROM_SETUP_DIR/populate.json \
    RULES_DIR=$PROM_SETUP_DIR/rules \
    SCRAPE_DIR=$PROM_SETUP_DIR/scrapes \
    ALERT_DIR=$PROM_SETUP_DIR/alerts \
    TEMPLATE_DIR=${PROM_SETUP_DIR}/templates 


# Create a non-root user and group for running Prometheus
RUN addgroup -S ${PROM_PID} && \
    adduser -S -D -H -s /sbin/nologin -G ${PROM_PID} ${PROM_PID}

USER ${PROM_PID}

# Set working directory
WORKDIR /prometheus

# Copy Prometheus binaries from builder stage
COPY --from=builder /usr/bin/prometheus /usr/bin/prometheus
COPY --from=builder /usr/bin/alertmanager /usr/bin/alertmanager

# Copy Python virtual environment from builder stage
COPY --from=builder /venv /venv
# Copy configuration files and directories, setting ownership
COPY ./prometheus.yml ${PROMETHEUS_CONFIG_FILE} 
COPY ./web.yml ${PROMETHEUS_WEB_CONFIG_FILE}
COPY ./rules ${RULES_DIR} 
COPY ./alerts ${ALERT_DIR} 
COPY ./scrapes ${SCRAPE_DIR} 
COPY ./templates  ${TEMPLATE_DIR}
COPY ./populate.json ${POPULATION_FILE} 
COPY ./prom-builder.py prom-builder.py 
COPY ./alertmanager.yml alertmanager.yml
COPY prometheus/prometheus.sh /prometheus.sh

USER root
# Make the Prometheus startup script executable
RUN chmod +x /prometheus.sh

# Expose the Prometheus port
EXPOSE ${PROM_PORT} ${ALERT_MANAGER_PORT}
# Define a healthcheck
HEALTHCHECK --interval=15s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:${PROM_PORT}/-/healthy && curl -f http://localhost:${ALERT_MANAGER_PORT}/-/healthy || exit 1
# Set the entrypoint
ENTRYPOINT ["/prometheus.sh"]