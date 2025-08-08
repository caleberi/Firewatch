# Firewatch
<img width="1431" height="876" alt="image" src="https://github.com/user-attachments/assets/62216fe9-5021-4abc-8711-645ad571b083" />

**Firewatch** is a robust Prometheus-based service monitoring and observability solution. Track metrics, visualise performance, and ensure your systems stay resilient with real-time insights. Light up your infrastructure's blind spots! 🔥

---

## What’s New?

- **TallyPort Metrics Server** (under `/tallyport`):  
  - Go-based HTTP service for custom Prometheus metric ingestion and exposure.
  - RESTful API endpoints `/init` (metric creation) & `/push` (metric updates).
  - Example React Native integration for mobile metric reporting.
  - Configurable via YAML, with clear Prometheus scrape instructions.
- **Dev Mode for Grafana:**  
  - Easily enable a Grafana+Prometheus stack for dashboard/plugin development using `DEV_MODE`.
  - Built-in plugin installation and image renderer support.
- **Enhanced Documentation:**  
  - Comprehensive usage and integration instructions.
  - Walkthroughs for both web and mobile metric pipelines.
- **Improved Docker & Build Scripts:**  
  - Containerised deployment for both core and dev stacks.
  - Streamlined setup for new contributors and users.

---

## Key Features

- **Prometheus Integration:** Seamlessly collect and aggregate metrics.
- **Real-Time Dashboards:** Visualise system health and trends.
- **Alerting:** Instant notifications for anomalies or outages.
- **Extensible Architecture:** Plug in exporters and third-party tools.
- **Resilient Monitoring:** Built for distributed, large-scale systems.
- **Historical Data Analysis:** Capacity planning and optimization.
- **User-Friendly Interface:** Intuitive web UI, now with extended endpoints for mobile/reporting integrations.

---

## Quick Start

1. **Clone the Repository**
   ```bash
   git clone https://github.com/caleberi/firewatch.git
   cd firewatch
   ```

2. **Set Up Prometheus**  
   Ensure Prometheus is running and Firewatch is configured as a scrape target.

3. **Install Dependencies & Start**
   - For Node.js:
     ```bash
     npm install
     npm start
     ```
   - With Docker:
     ```bash
     docker build -t firewatch .
     docker run -p 8080:8080 firewatch
     ```
   - For TallyPort (Go server):
     ```bash
     cd tallyport
     go build -o tallyport
     ./tallyport -config-file setting.yml
     ```

4. **(Optional) Dev Mode with Grafana**
   ```bash
   docker build --build-arg DEV_MODE=on -t firewatch-dev .
   docker run -e DEV_MODE=on -p 9090:9090 -p 3000:3000 firewatch-dev
   ```

---

## More Information

- **TallyPort Metrics Server:** See `/tallyport/README.md` for complete usage, API endpoints, and mobile integration examples.
- **Configuration:** Edit `config.yml` and `setting.yml` for custom setup.
- **Documentation:** See `/docs` and [GitHub Wiki](https://github.com/caleberi/firewatch/wiki).

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## License

This project is licensed under the [MIT License](LICENSE).

---

## Acknowledgements

- [Prometheus](https://prometheus.io/)
- [Grafana](https://grafana.com/)
- [PromQL-Anomaly-Detection (PAD)](https://github.com/grafana/promql-anomaly-detection)

---
