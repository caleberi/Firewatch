#!/bin/bash

build_tallyport() {
    log_message "Building Tallyport image: ${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would build ${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
    else
        docker buildx build -t "${TALLYPORT_IMAGE}:${VERSION_NUMBER}" \
        --build-arg TALLYPORT_PORT=8080 \
        --progress=plain -f tallyport/Dockerfile tallyport >> $LOG_FILE 2>&1 || {
            log_message "Error: Building ${TALLYPORT_IMAGE}:${VERSION_NUMBER} failed"
            exit 1
        }
    fi

    # Tag Tallyport image with 'latest'
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag ${TALLYPORT_IMAGE}:${VERSION_NUMBER} as ${TALLYPORT_IMAGE}:latest"
    else
        do_tag_image "${TALLYPORT_IMAGE}:${VERSION_NUMBER}" "${TALLYPORT_IMAGE}:latest"
    fi
}