# Firewatch
<img width="1431" height="876" alt="image" src="https://github.com/user-attachments/assets/62216fe9-5021-4abc-8711-645ad571b083" />



**Firewatch**, A robust Prometheus-based service monitoring and observability solution. Track metrics, visualize performance, and ensure your systems stay resilient with real-time insights. Light up your infrastructure's blind spots! 🔥**

---

## Features

- **Prometheus Integration:** Seamlessly collects and aggregates metrics using Prometheus.
- **Real-Time Dashboards:** Visualize system health, performance, and trends with customizable dashboards.
- **Alerting:** Get notified instantly of anomalies, outages, or threshold breaches.
- **Extensible Architecture:** Easily integrate with additional exporters and third-party tools.
- **Resilient Monitoring:** Built to monitor large-scale, distributed environments.
- **Historical Data Analysis:** Analyze trends over time for capacity planning and optimization.
- **User-Friendly Interface:** Intuitive web UI for both engineers and non-technical users.

---

## Getting Started

### Prerequisites

- [Prometheus](https://prometheus.io/) server
- Docker (optional, for containerised deployment)
- Node.js & npm (if building from source)
- Supported OS: Linux, macOS, Windows

### Installation

#### 1. Clone the Repository

```bash
git clone https://github.com/caleberi/firewatch.git
cd firewatch
```

#### 2. Configure Prometheus

Ensure your Prometheus server is running and accessible. Update your configuration to allow Firewatch to scrape necessary targets.

#### 3. Install Dependencies

```bash
npm install
```

#### 4. Start Firewatch

```bash
npm start
```

Or run with Docker:

```bash
docker build -t firewatch .
docker run -p 8080:8080 firewatch
```

---

## Configuration

Edit the `config.yml` file in the root directory to set up:

- Prometheus server URL
- Alerting channels (email, Slack, etc.)
- Custom dashboards and widgets

---

## Grafana Dev Mode

Firewatch supports a Grafana development mode for faster dashboard development and plugin testing.

### What is Dev Mode?

When the environment variable `DEV_MODE` is set to `on`, the container enables extra features for local development:
- **Grafana is automatically installed and launched** alongside Prometheus.
- **Plugin installation is easier** via the `GF_INSTALL_PLUGINS` environment variable.
- **Optional image renderer plugin** can be installed by setting `GF_INSTALL_IMAGE_RENDERER_PLUGIN=true`.
- **Healthchecks include Grafana,** so you know both Prometheus and Grafana are up.

### How to Use

**Run in Dev Mode with Docker:**
```bash
docker build --build-arg DEV_MODE=on -t firewatch-dev .
docker run -e DEV_MODE=on -p 9090:9090 -p 3000:3000 firewatch-dev
```
- Access Prometheus at [http://localhost:9090](http://localhost:9090)
- Access Grafana at [http://localhost:3000](http://localhost:3000) (default admin: `admin` / `admin`)

**Install custom plugins:**
```bash
docker run -e DEV_MODE=on -e GF_INSTALL_PLUGINS="grafana-clock-panel,grafana-simple-json-datasource" -p 3000:3000 firewatch-dev
```
**Enable the image renderer plugin:**
```bash
docker run -e DEV_MODE=on -e GF_INSTALL_IMAGE_RENDERER_PLUGIN=true -p 3000:3000 firewatch-dev
```

### Disabling Dev Mode

By default, dev mode is **off**. In production deployments, omit the `DEV_MODE` env or set it to `off` to disable Grafana and related features.

---

## Usage

- **Access the UI:** Open [http://localhost:8080](http://localhost:8080) in your browser.
- **View Dashboards:** Explore built-in and custom dashboards to monitor metrics in real time.
- **Set Alerts:** Configure alerting rules and notification channels.
- **Integrate Exporters:** Add new exporters as needed for custom metrics.

---

## Documentation

Full documentation is available in the `/docs` directory or at [GitHub Wiki](https://github.com/caleberi/firewatch/wiki).

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

This project is licensed under the [MIT License](LICENSE).

---

## Acknowledgements

- [Prometheus](https://prometheus.io/)
- [Grafana](https://grafana.com/) (for dashboard inspiration)
- [PromQL-Anomaly-Detection (PAD)](https://github.com/grafana/promql-anomaly-detection)
