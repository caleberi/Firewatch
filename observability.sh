#!/bin/sh
set -e

PROMETHEUS_EXECUTABLE_PROGRAM=/usr/bin/prometheus
PROMETHEUS_TOOL_EXECUTABLE_PROGRAM=/usr/bin/promtool
PROMETHEUS_CONFIG_FILE=/etc/prometheus/prometheus.yml
PROMETHEUS_WEB_CONFIG_FILE=/etc/prometheus/web.yml
POPULATION_FILE=/etc/prometheus/populate.json

RULES_DIR=/etc/prometheus/rules
SCRAPE_DIR=/etc/prometheus/scrapes


GRAFANA_PORT=${GRAFANA_PORT:-3000}
GRAFANA_EXECUTABLE_PROGRAM=/usr/share/grafana/grafana-server

PROM_PORT=${PROM_PORT:-9090} 
PROM_LOGLEVEL=${PROM_LOGLEVEL:-info} 
PROM_RETENTION_TIME=${PROM_RETENTION_TIME:-"30s"}
PROM_RETENTION_SIZE=${PROM_RETENTION_SIZE:-"512MB"}

DEV_MODE="${1:-off}"
GF_LOG_MODE="${2:-console}"
GF_PATHS_PLUGINS="${3:-/var/lib/grafana/plugins}"
GF_PATHS_DATA="${4:-/var/lib/grafana}"
GF_PATHS_LOGS="${5:-/var/log/grafana}"
GF_PATHS_PROVISIONING="${6:-/etc/grafana/provisioning}"

parse_cmd_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --GRAFANA_PORT)
                GRAFANA_PORT="$2"
                shift 2
                ;;
            --PROM_PORT)
                PROM_PORT="$2"
                shift 2
                ;;
            --PROM_LOGLEVEL)
                PROM_LOGLEVEL="$2"
                shift 2
                ;;
            --GF_PATHS_DATA)
                GF_PATHS_DATA="$2"
                shift 2
                ;;
            --GF_PATHS_PLUGINS)
                GF_PATHS_PLUGINS="$2"
                shift 2
                ;;
            --DEV_MODE)
                DEV_MODE="$2"
                shift 2
                ;;
            --GF_PATHS_PROVISIONING)
                GF_PATHS_PROVISIONING="$2"
                shift 2
                ;;
            --GF_PATHS_LOGS)
                GF_PATHS_LOGS="$2"
                shift 2
                ;;
            --GF_LOG_MODE)
                GF_LOG_MODE="$2"
                shift 2
                ;;
            --*)
                echo "Unknown option: $1" >&2
                shift 1
                ;;
            *)
                # Positional arg
                echo "Ignoring positional argument: $1"
                shift 1
                ;;
        esac
    done
}


parse_cmd_args "$@"

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


# Start Grafana as a daemon (in the background)
if [ "$DEV_MODE" = "on" ] && [ -x "$GRAFANA_EXECUTABLE_PROGRAM" ]; then
    echo "Starting Grafana as daemon..."
    echo "Grafana config file: /etc/grafana/grafana.ini"
    echo "Grafana homepath: /usr/share/grafana"
    echo "Grafana plugins dir: $GF_PATHS_PLUGINS"

    [ -f /etc/grafana/grafana.ini ] || { echo "Grafana config file not found"; exit 1; }

    nohup su -s /bin/sh grafana -c "
        cd /usr/share/grafana &&
        /usr/share/grafana/grafana server \
            --homepath=/usr/share/grafana \
            --config=/etc/grafana/grafana.ini \
            cfg:default.paths.plugins=\"$GF_PATHS_PLUGINS\" \
            cfg:default.paths.data=\"$GF_PATHS_DATA\" \
            cfg:default.paths.logs=\"$GF_PATHS_LOGS\" \
            cfg:default.paths.provisioning=\"$GF_PATHS_PROVISIONING\" \
            cfg:log.mode=\"$GF_LOG_MODE\" > /dev/null 2>&1 &"

    echo "Waiting for Grafana to start…"
    until curl -s "http://localhost:${GRAFANA_PORT:-3000}/api/health" >/dev/null; do
        sleep 1
    done
    echo "Grafana started"

    if [ "$DEV_MODE" = "on" ] && [ -n "$GF_INSTALL_PLUGINS" ]; then
        OLDIFS=$IFS
        IFS=','                   # split on commas
        set -e                    # exit on error
        for plugin in $GF_INSTALL_PLUGINS; do
            IFS=$OLDIFS           # restore default word‑splitting for each body
            if printf '%s\n' "$plugin" | grep -q ';'; then
                pluginUrl=${plugin%%;*}
                pluginInstallFolder=${plugin#*;}
                su -s /bin/sh grafana -c \
                    "/usr/share/grafana/grafana cli \
                     --pluginUrl \"$pluginUrl\" \
                     --pluginsDir \"$GF_PATHS_PLUGINS\" \
                     plugins install \"$pluginInstallFolder\""
            else
                su -s /bin/sh grafana -c \
                    "/usr/share/grafana/grafana cli \
                     --pluginsDir \"$GF_PATHS_PLUGINS\" plugins install \"$plugin\""
            fi
        done
        IFS=$OLDIFS
    fi
else
    echo "Skipping Grafana startup: DEV_MODE = $DEV_MODE or executable not found"
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
    --query.lookback-delta=5m \
    --query.timeout=2m \
    --enable-feature=memory-snapshot-on-shutdown,extra-scrape-metrics,promql-per-step-stats,promql-experimental-functions,concurrent-rule-eval,auto-reload-config,native-histograms,promql-duration-expr