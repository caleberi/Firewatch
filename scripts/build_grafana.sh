#!/bin/bash

# Grafana build configuration
GRAFANA_PLUGINS="grafana-clock-panel 1.0.1,grafana-simple-json-datasource 1.3.5"
GRAFANA_RENDER_PLUGIN="https://github.com/grafana/grafana-image-renderer/releases/latest/download/plugin-alpine-x64-no-chromium.zip"

build_grafana() {
    log_message "Building Grafana image: ${GRAFANA_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${GRAFANA_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${GRAFANA_IMAGE}:${VERSION_NUMBER}" \
        --build-arg BASE_OS_IMAGE=grafana/grafana \
        --build-arg BASE_OS_VERSION=12.0.2 \
        --build-arg GRAFANA_PORT=3000 \
        --build-arg GF_LOG_MODE=console \
        --build-arg GF_INSTALL_IMAGE_RENDERER_PLUGIN=true \
        --build-arg GF_INSTALL_MONGODB_DATASOURCE_PLUGIN=true \
        --build-arg GF_INSTALL_PLUGINS="${GRAFANA_PLUGINS}" \
        --build-arg GF_PLUGIN_IMAGE_RENDER_URL="${GRAFANA_RENDER_PLUGIN}" \
        --progress=plain -f grafana/Dockerfile . >> $LOG_FILE 2>&1 || {
            log_message "Error: Building ${GRAFANA_IMAGE}:${VERSION_NUMBER} failed"
            exit 1
        }
    fi

    # Tag Grafana image with 'latest'
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag ${GRAFANA_IMAGE}:${VERSION_NUMBER} as ${GRAFANA_IMAGE}:latest"
    else
        do_tag_image "${GRAFANA_IMAGE}:${VERSION_NUMBER}" "${GRAFANA_IMAGE}:latest"
    fi
}