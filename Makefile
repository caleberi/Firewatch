# Makefile for cudium-prometheus development

IMAGE_NAME = cudium-prometheus
BUILD_VERSION = 1.0.0
PROM_PORT = 9091

.PHONY: all
all: build run

# Build the Docker image
.PHONY: build
build:
	docker build -t $(IMAGE_NAME) --build-arg BUILD_VERSION=$(BUILD_VERSION) --build-arg PROM_PORT=$(PROM_PORT) .

# Run the Docker container
.PHONY: run
run:
	docker run -d --name $(IMAGE_NAME)-dev -p 127.0.0.1:9090:$(PROM_PORT) -e PROM_PORT=$(PROM_PORT) -e PROM_LOGLEVEL=debug $(IMAGE_NAME)

# Stop and remove the container
.PHONY: stop
stop:
	docker stop $(IMAGE_NAME)-dev || true
	docker rm $(IMAGE_NAME)-dev || true

# Clean up Docker images
.PHONY: clean
clean:
	docker rmi $(IMAGE_NAME) || true

# View container logs
.PHONY: logs
logs:
	docker logs $(IMAGE_NAME)-dev

# Check container health
.PHONY: health
health:
	curl -v http://127.0.0.1:9090/-/healthy

# Rebuild and rerun
.PHONY: rebuild
rebuild: stop clean build run