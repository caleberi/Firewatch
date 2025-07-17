#!/bin/bash
set -e

# Usage:
#   ./image_builder.sh [OPTIONS]
#
# Description:
#   This script builds and pushes Prometheus and Grafana Docker images to Docker Hub.
#   It supports version management, dry-run mode, multiple tags, and configuration via a build.conf file.
#
# Options:
#   -u, --username <username>   Docker Hub username (required)
#   -p, --password <password>   Docker Hub password (required)
#   -a, --account <account>     Docker Hub account/organization (required)
#   --dry-run                   Simulate operations without executing them
#   --increment=<type>          Increment version (major, minor, or patch)
#.  --no-push                   Avoid pushing to docker hub
#.  --rebuild                   Rebuild images 
#
# Examples:
#   ./image_builder.sh -u myuser -p mypass -a myaccount
#   ./image_builder.sh --username myuser --password mypass --account myaccount --increment=patch
#   ./image_builder.sh --dry-run -u myuser -p mypass -a myaccount
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
MAX_RETRIES=3
RETRY_DELAY=5
CONFIG_FILE="build.conf"
DRY_RUN=false
USE_BUILDX=false
NO_PUSH=false
REBUILD=false

# Default version numbers
MAJOR=1
MINOR=0
PATCH=0

log_message() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $1" | tee -a "$LOG_FILE"
}

cleanup() {
    log_message "Cleaning up temporary files..."
    rm -f *.log   # remove logfiles if uncommented
}

trap cleanup EXIT INT TERM

# Function to validate Docker credentials and account
validate_credentials() {
    local username="$1"
    local password="$2"
    local account="$3"
    
    if [[ -z "$username" || -z "$password" || -z "$account" ]]; then
        log_message "Error: Docker username, password, and account cannot be empty"
        exit 1
    fi
    
    if [[ ! "$username" =~ ^[a-zA-Z0-9_-]+$ ]] || [[ ! "$account" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log_message "Error: Invalid Docker username or account format"
        exit 1
    fi
}

validate_version() {
    local major="$1" minor="$2" patch="$3"
    if [[ ! "$major" =~ ^[0-9]+$ ]] || [[ ! "$minor" =~ ^[0-9]+$ ]] || [[ ! "$patch" =~ ^[0-9]+$ ]]; then
        log_message "Error: Version numbers must be non-negative integers (major: $major, minor: $minor, patch: $patch)"
        exit 1
    fi
}

increment_version() {
    local part="$1"
    case "$part" in
        major) ((MAJOR++)); MINOR=0; PATCH=0;;
        minor) ((MINOR++)); PATCH=0;;
        patch) ((PATCH++));;
        *) log_message "Error: Invalid version increment type"; exit 1;;
    esac
}

check_buildx() {
    if [ "$USE_BUILDX" = true ]; then
        if ! docker buildx version >/dev/null 2>&1; then
            log_message "Error: Docker Buildx is not installed or not available"
            exit 1
        fi
    fi
}

tag_image() {
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

push_image() {
    local image="$1"
    local retries=0
    
    while [ $retries -lt $MAX_RETRIES ]; do
        if [ "$DRY_RUN" = true ]; then
            log_message "DRY-RUN: Would push $image"
            return 0
        fi
        
        if docker push "$image" >> "push_$(basename $image).log" 2>&1; then
            log_message "Successfully pushed $image"
            return 0
        fi
        
        ((retries++))
        log_message "Failed to push $image (attempt $retries/$MAX_RETRIES)"
        sleep $RETRY_DELAY
    done
    
    log_message "Error: Failed to push $image after $MAX_RETRIES attempts"
    exit 1
}

if ! command -v docker >/dev/null 2>&1; then
    log_message "Error: Docker is not installed on this system"
    exit 1
fi

if [ -f "$CONFIG_FILE" ]; then
    log_message "Loading configuration from $CONFIG_FILE"
    source "$CONFIG_FILE"
fi

while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run) DRY_RUN=true; shift;;
        --no-push) NO_PUSH=true; shift;;
        --rebuild) REBUILD=true; shift;;
        --use-buildx) USE_BUILDX=true; shift;;
        --increment=*) increment_version "${1#*=}"; shift;;
        -u|--username) DOCKER_USERNAME="$2"; shift 2;;
        -p|--password) DOCKER_PASSWORD="$2"; shift 2;;
        -a|--account) DOCKER_ACCOUNT="$2"; shift 2;;
        *) log_message "Error: Unknown option $1"; exit 1;;
    esac
done

VERSION_TAG="${MAJOR}.${MINOR}.${PATCH}"
validate_version "$MAJOR" "$MINOR" "$PATCH"

check_buildx

# Image configurations
PROMETHEUS_IMAGE="${DOCKER_ACCOUNT}/prometheus"
GRAFANA_IMAGE="${DOCKER_ACCOUNT}/grafana"
GRAFANA_PLUGINS="grafana-clock-panel 1.0.1,grafana-simple-json-datasource 1.3.5"
GRAFANA_RENDER_PLUGIN="https://github.com/grafana/grafana-image-renderer/releases/latest/download/plugin-alpine-x64-no-chromium.zip"

# Check if Prometheus image exists
if [ "$DRY_RUN" = false ] && [ "$REBUILD" = false ]  && docker image ls -q "${PROMETHEUS_IMAGE}:${VERSION_TAG}" | grep -q .; then
    log_message "${PROMETHEUS_IMAGE}:${VERSION_TAG} already exists. Skipping build process."
    exit 0
fi

validate_build_context() {
    local context="$1"
    local dockerfile="$2"
    
    if [ ! -d "$context" ]; then
        log_message "Error: Build context directory $context does not exist"
        exit 1
    fi
    
    if [ ! -f "$dockerfile" ]; then
        log_message "Error: Dockerfile $dockerfile does not exist"
        exit 1
    fi
}

validate_build_context "./grafana" "./grafana/Dockerfile"
validate_build_context "./prometheus" "./prometheus/Dockerfile"

# Build Grafana image
log_message "Building Grafana image: ${GRAFANA_IMAGE}:${VERSION_TAG}"
if [ "$DRY_RUN" = true ]; then
    log_message "DRY-RUN: Would build ${GRAFANA_IMAGE}:${VERSION_TAG}"
else
    docker buildx build -t "${GRAFANA_IMAGE}:${VERSION_TAG}" \
    --build-arg BASE_OS_IMAGE=grafana/grafana \
    --build-arg BASE_OS_VERSION=12.0.2 \
    --build-arg GRAFANA_PORT=3000 \
    --build-arg GF_LOG_MODE=console \
    --build-arg GF_INSTALL_IMAGE_RENDERER_PLUGIN=true \
    --build-arg GF_INSTALL_MONGODB_DATASOURCE_PLUGIN=true \
    --build-arg GF_INSTALL_PLUGINS="${GRAFANA_PLUGINS}" \
    --build-arg GF_PLUGIN_IMAGE_RENDER_URL="${GRAFANA_RENDER_PLUGIN}" \
    --progress=plain -f grafana/Dockerfile . >> build_grafana.log 2>&1 || {
        log_message "Error: Building ${GRAFANA_IMAGE}:${VERSION_TAG} failed"
        exit 1
    }
fi

# Tag Grafana image with 'latest'
tag_image "${GRAFANA_IMAGE}:${VERSION_TAG}" "${GRAFANA_IMAGE}:latest"

# Build Prometheus image
log_message "Building Prometheus image: ${PROMETHEUS_IMAGE}:${VERSION_TAG}"
if [ "$DRY_RUN" = true ]; then
    log_message "DRY-RUN: Would build ${PROMETHEUS_IMAGE}:${VERSION_TAG}"
else
    docker buildx build -t "${PROMETHEUS_IMAGE}:${VERSION_TAG}" --no-cache \
    --build-arg PROM_PID=prometheus \
    --build-arg PROM_SETUP_DIR=/etc/prometheus \
    --build-arg PROM_PORT=9091 \
    --progress=plain -f prometheus/Dockerfile . >> build_prometheus.log 2>&1 || {
        log_message "Error: Building ${PROMETHEUS_IMAGE}:${VERSION_TAG} failed"
        exit 1
    }
fi

# Tag Prometheus image with 'latest'
tag_image "${PROMETHEUS_IMAGE}:${VERSION_TAG}" "${PROMETHEUS_IMAGE}:latest"

if [ "$NO_PUSH" != true ]; then
    validate_credentials "$DOCKER_USERNAME" "$DOCKER_PASSWORD" "$DOCKER_ACCOUNT"
    # Login to Docker Hub
    if [ "$DRY_RUN" = true ]; then
        log_message "DRY-RUN: Would login to Docker Hub"
    else
        log_message "Logging in to Docker Hub"
        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin >> docker_login.log 2>&1 || {
            log_message "Error: Docker Hub authentication failed"
            exit 1
        }
    fi

    # Push images to Docker Hub
    push_image "${GRAFANA_IMAGE}:${VERSION_TAG}"
    push_image "${GRAFANA_IMAGE}:latest"
    push_image "${PROMETHEUS_IMAGE}:${VERSION_TAG}"
    push_image "${PROMETHEUS_IMAGE}:latest"

fi

log_message "Successfully built, tagged, and pushed images:"
log_message "*) ${GRAFANA_IMAGE}:${VERSION_TAG}"
log_message "*) ${GRAFANA_IMAGE}:latest"
log_message "*) ${PROMETHEUS_IMAGE}:${VERSION_TAG}"
log_message "*) ${PROMETHEUS_IMAGE}:latest"