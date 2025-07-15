ARG BASE_OS_IMAGE=grafana/grafana
ARG BASE_OS_VERSION=main

# Use the specified base image and version, naming this stage 'grafana-builder'
FROM ${BASE_OS_IMAGE}:${BASE_OS_VERSION} AS grafana-builder

# Add metadata labels for maintainer and source repository
LABEL maintainer="Firewatch Labs <hello@firewatch.com>"
LABEL org.opencontainers.image.title="Visualization Service"
LABEL org.opencontainers.image.source="https://github.com/caleberi/firewatch"
LABEL org.opencontainers.image.description="Visualization service for Configured applications"
LABEL org.opencontainers.image.vendor="Firewatch"
LABEL com.firewatch.service="service-watcher"

# Set environment variables for Grafana paths and configurations
ENV PATH="/usr/share/grafana/bin:$PATH" \                          
    GF_PATHS_PLUGINS="/var/lib/grafana-plugins" \               
    GF_PLUGIN_RENDERING_CHROME_BIN="/usr/bin/chrome" \   
    GF_PATHS_CONFIG="/etc/grafana/grafana.ini" \  
    GF_PATHS_DATA="/var/lib/grafana" \   
    GF_PATHS_HOME="/usr/share/grafana" \   
    GF_PATHS_LOGS="/var/log/grafana" \                          
    GF_PATHS_PLUGINS="/var/lib/grafana/plugins" \               
    GF_PATHS_PROVISIONING="/etc/grafana/provisioning"

# Define build arguments for Grafana configuration
ARG GRAFANA_PORT=3000                                              # Default port for Grafana
ARG GF_LOG_MODE="console"                                          # Logging mode for Grafana
ARG GF_INSTALL_IMAGE_RENDERER_PLUGIN="false"                       # Flag to install image renderer plugin
ARG GRAFANA_EXECUTABLE_PROGRAM=/usr/share/grafana/bin/grafana      # Path to Grafana executable
ARG GF_INSTALL_PLUGINS="grafana-clock-panel 1.0.1,grafana-simple-json-datasource 1.3.5"  # List of plugins to install
ARG ENV GF_PLUGIN_IMAGE_RENDER_URL="https://github.com/grafana/grafana-image-renderer/releases/latest/download/plugin-alpine-x64-no-chromium.zip"  # URL for image renderer plugin

# Install the image renderer plugin if GF_INSTALL_IMAGE_RENDERER_PLUGIN is true and running on Alpine
RUN if [ "$GF_INSTALL_IMAGE_RENDERER_PLUGIN" = "true" ]; then \
    # Check if the OS is Alpine by searching /etc/issue
    if grep -i -q alpine /etc/issue; then \
    # Install dependencies (udev, fonts, and Chromium) for Alpine
    apk add --no-cache udev ttf-opensans chromium && \
    # Create a symlink for Chromium to the expected Chrome binary path
    ln -s /usr/bin/chromium-browser "$GF_PLUGIN_RENDERING_CHROME_BIN" && \
    # Install the image renderer plugin using Grafana CLI
    su -s /bin/sh -c "$(which grafana) cli --pluginsDir \"$GF_PATHS_PLUGINS\" --pluginUrl \"$GF_PLUGIN_IMAGE_RENDER_URL\" plugins install grafana-image-renderer"; \
    else \
    # Print error if trying to install on a non-Alpine OS
    echo "Cannot install image render plugin on non-alpine OS"; \
    fi; \
    fi;

# Install specified Grafana plugins from GF_INSTALL_PLUGINS
RUN if [ -n "$GF_INSTALL_PLUGINS" ]; then \
    # Save the current Internal Field Separator (IFS) to restore later
    OLDIFS=$IFS; \
    # Set IFS to comma for splitting plugin list
    IFS=','; \
    # Enable exit-on-error for the plugin installation loop
    set -e; \
    # Loop through each plugin entry in GF_INSTALL_PLUGINS
    for plugin_entry in $GF_INSTALL_PLUGINS; do \
    # Restore original IFS for each iteration
    IFS=$OLDIFS; \
    # Extract plugin ID (first field)
    plugin_id=$(echo "$plugin_entry" | awk '{print $1}'); \
    # Extract plugin version (second field, if available)
    plugin_version=$(echo "$plugin_entry" | awk '{print $2}'); \
    # Check if a version is specified
    if [ -n "$plugin_version" ]; then \
    # Install plugin with specified version
    grafana cli --pluginsDir "$GF_PATHS_PLUGINS" plugins install "$plugin_id" "$plugin_version"; \
    else \
    # Install latest version of plugin if no version specified
    grafana cli --pluginsDir "$GF_PATHS_PLUGINS" plugins install "$plugin_id"; \
    fi; \
    done; \
    # Restore original IFS after loop
    IFS=$OLDIFS; \
    fi

# Copy custom Grafana configuration file to the container
COPY grafana/grafana.ini ${GF_PATHS_CONFIG}
# Copy the startup script (grafana.sh) to the container
COPY grafana/grafana.sh /grafana.sh

# Switch to root user to set permissions
USER root
# Make the startup script executable
RUN chmod +x /grafana.sh
# Switch to the grafana user for running the container
USER grafana

# Expose the Grafana port (default 3000)
EXPOSE ${GRAFANA_PORT}

# Define a health check to verify Grafana is running
HEALTHCHECK --interval=15s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:${GRAFANA_PORT}/api/health || exit 1

# Set the entrypoint to the startup script
ENTRYPOINT ["/grafana.sh"]