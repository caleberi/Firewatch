#!/bin/sh
# Specifies that this is a shell script to be executed by the Bourne shell (sh)

set -e
# Enables the script to exit immediately if any command fails (non-zero exit status)

# Store the path to the Grafana executable by using the 'which' command
GRAFANA_EXECUTABLE_PROGRAM=$(which grafana)

# Check if the Grafana executable exists and is executable
if [ -x "$GRAFANA_EXECUTABLE_PROGRAM" ]; then
    # If Grafana executable is found, print startup message
    echo "Starting Grafana..."
    # Display the path to the Grafana configuration file
    echo "Grafana config file: /etc/grafana/grafana.ini"
    # Display the Grafana home directory path
    echo "Grafana homepath: /usr/share/grafana"
    # Display the directory for Grafana plugins, using the environment variable GF_PATHS_PLUGINS
    echo "Grafana plugins dir: $GF_PATHS_PLUGINS"

    # Check if the Grafana configuration file exists
    [ -f /etc/grafana/grafana.ini ] || { echo "Grafana config file not found"; exit 1; }
    # If the config file exists, execute the Grafana server with specified options
    exec $GRAFANA_EXECUTABLE_PROGRAM server \
        --homepath=/usr/share/grafana \                     # Set the Grafana home directory
        --config=/etc/grafana/grafana.ini \                 # Specify the configuration file
        cfg:default.paths.plugins="$GF_PATHS_PLUGINS" \     # Set the plugins directory
        cfg:default.paths.data="$GF_PATHS_DATA" \           # Set the data directory
        cfg:default.paths.logs="$GF_PATHS_LOGS" \           # Set the logs directory
        cfg:default.paths.provisioning="$GF_PATHS_PROVISIONING" \ # Set the provisioning directory
        cfg:log.mode="$GF_LOG_MODE"                         # Set the logging mode
else
    # If the Grafana executable is not found, print an error message and exit
    echo "Skipping Grafana startup: executable not found"
    exit 1
fi