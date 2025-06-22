
# Firewatch

**Firewatch** is a robust, Prometheus-based service monitoring and observability solution. Designed to help you track metrics, visualize performance, and ensure your systems remain resilient, firewatch provides real-time insights to illuminate your infrastructure’s blind spots.

🔥 **Light up your infrastructure’s blind spots!** 🔥

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
- Docker (optional, for containerized deployment)
- Node.js & npm (if building from source)
- Supported OS: Linux, macOS, Windows

### Installation

#### 1. Clone the Repository

```bash
git clone https://github.com/caleberi/firewatch.git
cd firewatch
```

#### 2. Configure Prometheus

Ensure your Prometheus server is running and accessible. Update your configuration to allow firewatch to scrape necessary targets.

#### 3. Install Dependencies

```bash
npm install
```

#### 4. Start firewatch

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
- Community contributors

---

## Contact

For support or questions, please open an [issue](https://github.com/caleberi/firewatch/issues) or contact [caleberi](https://github.com/caleberi).
