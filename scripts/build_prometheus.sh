#!/bin/bash

build_prometheus() {
    log_message "Building Prometheus image: ${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"\
        --build-arg PROM_PID=prometheus \
        --build-arg PROM_SETUP_DIR=/etc/prometheus \
        --build-arg PROM_PORT=9091 \
        --progress=plain -f prometheus/Dockerfile . >> $LOG_FILE 2>&1 || {
            log_message "Error: Building ${PROMETHEUS_IMAGE}:${VERSION_NUMBER} failed"
            exit 1
        }
    fi

    # Tag Prometheus image with 'latest'
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag ${PROMETHEUS_IMAGE}:${VERSION_NUMBER} as ${PROMETHEUS_IMAGE}:latest"
    else
        do_tag_image "${PROMETHEUS_IMAGE}:${VERSION_NUMBER}" "${PROMETHEUS_IMAGE}:latest"
    fi
}