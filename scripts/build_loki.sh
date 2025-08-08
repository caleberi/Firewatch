#!/bin/bash

build_loki() {
    log_message "Building Loki image: ${LOKI_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${LOKI_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${LOKI_IMAGE}:${VERSION_NUMBER}" \
        --progress=plain -f loki/Dockerfile ./loki >> "$LOG_FILE" 2>&1 || {
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