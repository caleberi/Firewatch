#!/bin/bash

log_message() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $1" | tee -a "$LOG_FILE"
}

log_fatal() {
    log_message "$1"
    exit 1
}

do_logfile_cleanup() {
    log_message "Cleaning up temporary files..."
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would clean up temporary files (*.log)"
        return 0
    fi
    rm -f *.log 2>/dev/null 
}


do_check_program_and_file() {
    local target="$1"
    if ! command -v "$target" >/dev/null 2>&1 && [ ! -x "$target" ]; then
        log_fatal "Error: $target is neither a program in PATH nor an executable file"
    fi
}

do_check_docker_buildx() {
    do_check_program_and_file docker
    if ! docker buildx version >/dev/null 2>&1; then
        log_fatal "Error: Docker Buildx is not available"
    fi
}

do_validate_version() {
    local version="$1"
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_fatal "Error: Version '$version' does not match MAJOR.MINOR.PATCH format"
    fi
    local major="${version%%.*}"
    local rest="${version#*.}"
    local minor="${rest%%.*}"
    local patch="${rest#*.}"
    if [[ ! "$major" =~ ^[0-9]+$ || ! "$minor" =~ ^[0-9]+$ || ! "$patch" =~ ^[0-9]+$ ]]; then
        log_fatal "Error: Version '$version' contains invalid numbers in MAJOR, MINOR, or PATCH"
    fi
}

validate_build_context() {
    local context="$1"
    local dockerfile="$2"
    
    if [ ! -d "$context" ]; then
        log_fatal "Error: Build context directory $context does not exist"
    fi
    
    if [ ! -f "$dockerfile" ]; then
        log_fatal "Error: Dockerfile $dockerfile does not exist"
    fi
}

do_check_build_context() {
    case $1 in
        prometheus|PROMETHEUS) validate_build_context "./prometheus" "./prometheus/Dockerfile";;
        grafana|GRAFANA) validate_build_context "./grafana" "./grafana/Dockerfile";;
        tallyport|TALLYPORT) validate_build_context "./tallyport" "./tallyport/Dockerfile";;
        loki|LOKI) validate_build_context "./loki" "./loki/Dockerfile";;
        promtail|PROMTAIL) validate_build_context "./promtail" "./promtail/Dockerfile";;
        *) log_fatal "Error: Unknown build context '$1'";;
    esac
}

do_check_docker_image() {
    local image="$1"
    if [ -z "$image" ]; then
        log_fatal "Error: No Docker image name provided"
    fi
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would check for existing Docker image '$image'"
        return 0
    fi
    if docker images -q "$image" | grep -q .; then
        log_fatal "Error: Docker image '$image' exist locally"
    fi
}


do_tag_image() {
    local image="$1"
    local tag="$2"
    if [ "$DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would tag $image as $tag"
        return 0
    fi
    
    if docker tag "$image" "$tag" 2>&1; then
        log_message "Successfully tagged $image as $tag"
        return 0
    else
        log_message "Error: Failed to tag $image as $tag"
        exit 1
    fi
}
