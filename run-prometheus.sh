#!/bin/sh

PROMETHEUS_EXECUTABLE_PROGRAM=/usr/bin/prometheus
PROMETHEUS_TOOL_EXECUTABLE_PROGRAM=/usr/bin/promtool
PROMETHEUS_CONFIG_FILE=/etc/prometheus/prometheus.yml
PROMETHEUS_WEB_CONFIG_FILE=/etc/prometheus/web.yml
POPULATION_FILE=/etc/prometheus/populate.json

RULES_DIR=/etc/prometheus/rules
SCRAPE_DIR=/etc/prometheus/scrapes

PROM_PORT=${PROM_PORT:-9090} 
PROM_LOGLEVEL=${PROM_LOGLEVEL:-info} 
PROM_RETENTION_TIME=${PROM_RETENTION_TIME:-"30s"}
PROM_RETENTION_SIZE=${PROM_RETENTION_SIZE:-"512MB"}


if [ ! -x "$PROMETHEUS_EXECUTABLE_PROGRAM" ]; then
    echo "Error: Prometheus executable not found or not executable at $PROMETHEUS_EXECUTABLE_PROGRAM"
    exit 1
fi

if [ ! -x "$PROMETHEUS_TOOL_EXECUTABLE_PROGRAM" ]; then
    echo "Error: Promtool executable not found or not executable at $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM"
    exit 1
fi

if [ -f "$PROMETHEUS_WEB_CONFIG_FILE" ]; then
    echo "Processing web config: $PROMETHEUS_WEB_CONFIG_FILE with prom-builder.py..."
    source /venv/bin/activate
    python3 prom-builder.py "$PROMETHEUS_WEB_CONFIG_FILE" "-"
    if [ $? -eq 0 ]; then
        echo "Successfully processed $PROMETHEUS_WEB_CONFIG_FILE"
    else
        echo "Error: Failed to process $PROMETHEUS_WEB_CONFIG_FILE"
    fi
    $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM check web-config "$PROMETHEUS_WEB_CONFIG_FILE" || {
        echo "Error: Web configuration check failed for $PROMETHEUS_WEB_CONFIG_FILE"
        exit 1
    }
else
    echo "Warning: Web configuration file $PROMETHEUS_WEB_CONFIG_FILE not found, skipping check"
fi

if [ -d "$RULES_DIR" ] && [ "$(ls -A "$RULES_DIR"/*.yml 2>/dev/null)" ]; then
    $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM check rules "$RULES_DIR"/*.yml || {
        echo "Error: Rules check failed for $RULES_DIR"
        exit 1
    }
else
    echo "Warning: No .yml files found in rules directory $RULES_DIR, skipping check"
fi

if [ -z "$SCRAPE_DIR" ]; then
    echo "Error: SCRAPE_DIR environment variable is not set."
    exit 1
fi

# Check if SCRAPE_DIR exists and contains .yml files
if [ -d "$SCRAPE_DIR" ] && ls "$SCRAPE_DIR"/*.yml >/dev/null 2>&1; then
    # Loop through all .yml files in SCRAPE_DIR
    for yaml_file in "$SCRAPE_DIR"/*.yml; do
        # Run prom-builder.py on each YAML file to replace environment variable placeholders
        # Example: Resolves placeholders like ${TARGET_HOST} in prometheus.yml
        echo "Processing $yaml_file with prom-builder.py..."
        python prom-builder.py "$yaml_file" "-"
        if [ $? -eq 0 ]; then
            echo "Successfully processed $yaml_file"
        else
            echo "Error: Failed to process $yaml_file"
        fi
        $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM check config --syntax-only --lint=all --lint-fatal --ignore-unknown-fields "$yaml_file" || {
            echo "Error: Configuration check failed for $yaml_file"
            cat $yaml_file
            exit 1
        }
    done
else
    # Output warning if no .yml files are found or directory doesn’t exist
    echo "Warning: No .yml files found in scrapes directory $SCRAPE_DIR, skipping environment resolution"
fi


echo "Working with Prometheus executable located in:"
echo ">> $PROMETHEUS_EXECUTABLE_PROGRAM"
echo ">> $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM"

echo "Running lint checks on configuration files"

if [ -f "$PROMETHEUS_CONFIG_FILE" ]; then
    echo "Processing $PROMETHEUS_CONFIG_FILE with prom-builder.py..."
    source /venv/bin/activate
    if [ -f "$POPULATION_FILE" ]; then
        python3 prom-builder.py "$PROMETHEUS_CONFIG_FILE" "$POPULATION_FILE"
    else 
        python3 prom-builder.py "$PROMETHEUS_CONFIG_FILE" "-"
    fi
    if [ $? -eq 0 ]; then
        echo "Successfully processed $PROMETHEUS_CONFIG_FILE"
    else
        echo "Error: Failed to process $PROMETHEUS_CONFIG_FILE"
    fi
    $PROMETHEUS_TOOL_EXECUTABLE_PROGRAM check config --syntax-only --lint=all --lint-fatal --ignore-unknown-fields "$PROMETHEUS_CONFIG_FILE" || {
        echo "Error: Configuration check failed for $PROMETHEUS_CONFIG_FILE"
        cat $PROMETHEUS_CONFIG_FILE
        exit 1
    }
    echo "<<<< View Final $PROMETHEUS_CONFIG_FILE >>>>>"
    cat $PROMETHEUS_CONFIG_FILE 
else
    echo "Error: Configuration file $PROMETHEUS_CONFIG_FILE not found"
    exit 1
fi

echo "Starting Prometheus With $PROMETHEUS_CONFIG_FILE..."
exec $PROMETHEUS_EXECUTABLE_PROGRAM \
    --enable-feature=exemplar-storage \
    --config.file=$PROMETHEUS_CONFIG_FILE \
    --storage.tsdb.retention.time="$PROM_RETENTION_TIME"\
    --storage.tsdb.retention.size="$PROM_RETENTION_SIZE"\
    --storage.tsdb.wal-compression \
    --storage.tsdb.path=/prometheus \
    --web.listen-address=0.0.0.0:$PROM_PORT \
    --web.enable-lifecycle \
    --web.enable-admin-api \
    --web.config.file="$PROMETHEUS_WEB_CONFIG_FILE" \
    --auto-gomemlimit.ratio=0.85 \
    --log.level=$PROM_LOGLEVEL \
    --log.format=json \
    --enable-feature=memory-snapshot-on-shutdown,extra-scrape-metrics,promql-per-step-stats,promql-experimental-functions,concurrent-rule-eval,auto-reload-config,native-histograms,promql-duration-expr