#!/bin/bash

LOKI_BASE_IMAGE="grafana/loki"
LOKI_BASE_VERSION="3.2.0"
LOKI_PORT=3100
LOKI_CONFIG_FILE="loki-config.yaml"

build_loki() {
    log_message "Building Loki image: ${LOKI_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${LOKI_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${LOKI_IMAGE}:${VERSION_NUMBER}" \
        --build-arg BASE_OS_IMAGE="${LOKI_BASE_IMAGE}" \
        --build-arg BASE_OS_VERSION="${LOKI_BASE_VERSION}" \
        --build-arg LOKI_PORT="${LOKI_PORT}" \
        --build-arg LOKI_CONFIG_FILE="${LOKI_CONFIG_FILE}" \
        --progress=plain -f loki/Dockerfile . >> "$LOG_FILE" 2>&1 || {
            log_message "Error: Building ${LOKI_IMAGE}:${VERSION_NUMBER} failed"
            exit 1
        }
    fi

    # Tag Loki image with 'latest'
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag ${LOKI_IMAGE}:${VERSION_NUMBER} as ${LOKI_IMAGE}:latest"
    else
        do_tag_image "${LOKI_IMAGE}:${VERSION_NUMBER}" "${LOKI_IMAGE}:latest"
    fi
}