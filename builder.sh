#!/bin/bash
set -e

# Usage:
#   ./image_builder.sh [OPTIONS]
#
# Description:
#   This script builds and pushes Prometheus, Grafana, and/or Tallyport Docker images to Docker Hub.
#   It supports version management, dry-run mode, multiple tags, and configuration via a build.conf file.
#   Use --image to build a specific image (grafana, prometheus, or tallyport).
#
# Options:
#   -u, --username <username>   Docker Hub username (required)
#   -p, --password <password>   Docker Hub password (required)
#   -a, --account  <account>    Docker Hub account/organization (required)
#   -v, --version  <version>    Version Tag For Build (required)
#   --image=<value>             Set Image Value (grafana, prometheus, tallyport, or all)
#   --no-push                   Avoid pushing to docker hub
#   --rebuild                   Rebuild images 
#   --dry-run                   Simulate operations without executing them
# Examples:
#   ./image_builder.sh -u myuser -p mypass -a myaccount --image=grafana
#   ./image_builder.sh --username myuser --password mypass --account myaccount --version=1.0.1 --image=prometheus
#   ./image_builder.sh --dry-run -u myuser -p mypass -a myaccount --image=tallyport
#
# Notes:
#   - Version numbers are defined in the script or build.conf (MAJOR.MINOR.PATCH)
#   - The script checks for existing images before building
#   - Logs are written to build_YYYYMMDD_HHMMSS.log and specific operation logs
#   - Temporary files are cleaned up on exit
#   - Docker must be installed and running
#   - Configuration can be set in build.conf file

# Configuration
LOG_FILE="build_$(date +%Y%m%d_%H%M%S).log"
MAX_RETRIES_PER_IMAGE=3
USE_DOCKER_BUILDX=false
USE_VERSION_MANAGEMENT=false
TRIGGER_IMAGE_PUSH=false
TRIGGER_IMAGE_REBUILD=false
RETRY_DELAY_TIME_IN_SECONDS=5
SIMULATE_DRY_RUN=false
IMAGE_NAME="all"

do_parse_input() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --image=*) IMAGE_NAME="${1#*=}"; shift;;
            --dry-run) SIMULATE_DRY_RUN=true; shift;;
            --no-push) TRIGGER_IMAGE_PUSH=true; shift;;
            --rebuild) TRIGGER_IMAGE_REBUILD=true; shift;;
            --use-buildx) USE_DOCKER_BUILDX=true; shift;;
            -v|--version) 
                VERSION_NUMBER="$2"; 
                USE_VERSION_MANAGEMENT=true; 
                shift 2;;
            -u|--username) DOCKER_USERNAME="$2"; shift 2;;
            -p|--password) DOCKER_PASSWORD="$2"; shift 2;;
            -a|--account) DOCKER_ACCOUNT="$2"; shift 2;;
            *) log_fatal "Error: Unknown option $1"; exit 1;;
        esac
    done
}


source ./scripts/prerequisites.sh
source ./scripts/build_loki.sh
source ./scripts/build_grafana.sh
source ./scripts/build_prometheus.sh
source ./scripts/build_tallyport.sh
source ./scripts/build_promtail.sh


do_parse_input "$@"
do_validate_version "$VERSION_NUMBER"
trap do_logfile_cleanup EXIT INT TERM
do_check_docker_buildx

# Image configurations
LOKI_IMAGE="${DOCKER_ACCOUNT}/loki"
GRAFANA_IMAGE="${DOCKER_ACCOUNT}/grafana"
TALLYPORT_IMAGE="${DOCKER_ACCOUNT}/tallyport"
PROMETHEUS_IMAGE="${DOCKER_ACCOUNT}/prometheus"
PROMTAIL_IMAGE="${DOCKER_ACCOUNT}/promtail"


# Validate build context and check existing images based on IMAGE_NAME
case $IMAGE_NAME in
    grafana|GRAFANA)
        do_check_build_context grafana
        do_check_docker_image "$GRAFANA_IMAGE"
        ;;
    prometheus|PROMETHEUS)
        do_check_build_context prometheus
        do_check_docker_image "$PROMETHEUS_IMAGE"
        ;;
    tallyport|TALLYPORT)
        do_check_build_context tallyport
        do_check_docker_image "$TALLYPORT_IMAGE"
        ;;
    loki|LOKI)
        do_check_build_context loki
        do_check_docker_image "$LOKI_IMAGE"
        ;;
    promtail|PROMTAIL)
        do_check_build_context promtail
        do_check_docker_image "$PROMTAIL_IMAGE"
        ;;
    all|ALL)
        do_check_build_context grafana
        do_check_build_context prometheus
        do_check_build_context tallyport
        do_check_build_context loki
        do_check_build_context promtail
        do_check_docker_image "$GRAFANA_IMAGE"
        do_check_docker_image "$TALLYPORT_IMAGE"
        do_check_docker_image "$PROMETHEUS_IMAGE"
        do_check_docker_image "$LOKI_IMAGE"
         do_check_docker_image "$PROMTAIL_IMAGE"
        ;;
    *) log_fatal "Error: Invalid image name '$IMAGE_NAME'. Must be 'grafana', 'prometheus', 'tallyport', or 'all'";;
esac

# Execute build functions based on IMAGE_NAME
case $IMAGE_NAME in
    grafana|GRAFANA)
        build_grafana
        ;;
    prometheus|PROMETHEUS)
        build_prometheus
        ;;
    tallyport|TALLYPORT)
        build_tallyport
        ;;
    loki|LOKI)
        build_loki
        ;;
    promtail|PROMTAIL)
        build_promtail
        ;;
    all|ALL)
        build_grafana
        build_prometheus
        build_tallyport
        build_loki
        build_promtail
        ;;
esac

if [ "$TRIGGER_IMAGE_PUSH" != true ]; then
    validate_credentials "$DOCKER_USERNAME" "$DOCKER_PASSWORD" "$DOCKER_ACCOUNT"
    # Login to Docker Hub
    if [ "$SIMULATE_DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would login to Docker Hub"
    else
        log_message "Logging in to Docker Hub"
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin >> docker_login.log 2>&1 || {
            log_message "Error: Docker Hub authentication failed"
            exit 1
        }
    fi

    # Push images to Docker Hub based on IMAGE_NAME
    case $IMAGE_NAME in
        grafana|GRAFANA)
            push_image "${GRAFANA_IMAGE}:${VERSION_NUMBER}"
            push_image "${GRAFANA_IMAGE}:latest"
            ;;
        prometheus|PROMETHEUS)
            push_image "${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
            push_image "${PROMETHEUS_IMAGE}:latest"
            ;;
        tallyport|TALLYPORT)
            push_image "${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
            push_image "${TALLYPORT_IMAGE}:latest"
            ;;
        loki|LOKI)
            push_image "${LOKI_IMAGE}:${VERSION_NUMBER}"
            push_image "${LOKI_IMAGE}:latest"
            ;;
        promtail|PROMTAIL)
            push_image "${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
            push_image "${PROMTAIL_IMAGE}:latest"
            ;;
        all|ALL)
            push_image "${GRAFANA_IMAGE}:${VERSION_NUMBER}"
            push_image "${GRAFANA_IMAGE}:latest"
            push_image "${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
            push_image "${PROMETHEUS_IMAGE}:latest"
            push_image "${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
            push_image "${TALLYPORT_IMAGE}:latest"
            push_image "${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
            push_image "${PROMTAIL_IMAGE}:latest"
            push_image "${LOKI_IMAGE}:${VERSION_NUMBER}"
            push_image "${LOKI_IMAGE}:latest"
            ;;
    esac
fi

# Log success message based on IMAGE_NAME
log_message "Successfully built, tagged, and pushed images:"
case $IMAGE_NAME in
    grafana|GRAFANA)
        log_message "*) ${GRAFANA_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${GRAFANA_IMAGE}:latest"
        ;;
    prometheus|PROMETHEUS)
        log_message "*) ${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${PROMETHEUS_IMAGE}:latest"
        ;;
    tallyport|TALLYPORT)
        log_message "*) ${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${TALLYPORT_IMAGE}:latest"
        ;;
    loki|LOKI)
        log_message "*) ${LOKI_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${LOKI_IMAGE}:latest"
        ;;
    promtail|PROMTAIL)
        log_message "*) ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${PROMTAIL_IMAGE}:latest"
        ;;
    all|ALL)
        log_message "*) ${GRAFANA_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${GRAFANA_IMAGE}:latest"
        log_message "*) ${PROMETHEUS_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${PROMETHEUS_IMAGE}:latest"
        log_message "*) ${TALLYPORT_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${TALLYPORT_IMAGE}:latest"
        log_message "*) ${LOKI_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${LOKI_IMAGE}:latest"
        log_message "*) ${PROMTAIL_IMAGE}:${VERSION_NUMBER}"
        log_message "*) ${PROMTAIL_IMAGE}:latest"
        ;;
esac