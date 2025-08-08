#!/bin/bash


build_promtail() {
    log_message "Building Promtail image: ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${PROMTAIL_IMAGE}:${VERSION_NUMBER}" \
        --progress=plain -f promtail/Dockerfile ./promtail >> "$LOG_FILE" 2>&1 || {
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