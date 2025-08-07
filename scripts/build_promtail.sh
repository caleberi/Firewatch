#!/bin/bash

# Promtail build configuration
PROMTAIL_BASE_IMAGE="grafana/promtail"
PROMTAIL_BASE_VERSION="3.2.0"
PROMTAIL_PORT=9080
PROMTAIL_CONFIG_FILE="promtail-config.yaml"

build_promtail() {
    log_message "Building Promtail image: ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${PROMTAIL_IMAGE}:${VERSION_NUMBER}" \
        --build-arg BASE_OS_IMAGE="${PROMTAIL_BASE_IMAGE}" \
        --build-arg BASE_OS_VERSION="${PROMTAIL_BASE_VERSION}" \
        --build-arg PROMTAIL_PORT="${PROMTAIL_PORT}" \
        --build-arg PROMTAIL_CONFIG_FILE="${PROMTAIL_CONFIG_FILE}" \
        --progress=plain -f promtail/Dockerfile . >> "$LOG_FILE" 2>&1 || {
            log_message "Error: Building ${PROMTAIL_IMAGE}:${VERSION_NUMBER} failed"
            exit 1
        }
    fi

    # Tag Promtail image with 'latest'
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag ${PROMTAIL_IMAGE}:${VERSION_NUMBER} as ${PROMTAIL_IMAGE}:latest"
    else
        do_tag_image "${PROMTAIL_IMAGE}:${VERSION_NUMBER}" "${PROMTAIL_IMAGE}:latest"
    fi
}