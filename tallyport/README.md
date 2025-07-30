# TallyPort Metrics Server

TallyPort is a Go-based HTTP server designed to collect and expose custom Prometheus metrics. It provides two primary endpoints: `/init` to create new metrics and `/push` to update metric values. The server uses the Prometheus client library to register and expose metrics, which can be scraped by a Prometheus instance at the `/metrics` endpoint. This README explains how to set up the server and integrate it with a React Native application to send metrics.

## Features
- Supports Prometheus metric types: Counter, Gauge, Histogram, and Summary.
- RESTful API with `/init` and `/push` endpoints for metric creation and updates.
- Configurable via a YAML file (`setting.yml`).
- Thread-safe metric storage using `sync.RWMutex`.
- Built with `go-chi` for routing and `zerolog` for logging.
- Exposes metrics at `/metrics` for Prometheus scraping.

## Prerequisites
- **Go**: Version 1.18 or higher.
- **Prometheus**: A running Prometheus instance to scrape metrics.
- **React Native**: A React Native development environment for the client application.
- **Node.js**: For React Native dependencies and setup.
- **Git**: For cloning the repository.

## Setup Instructions

### 1. Clone the Repository
```bash
git clone <repository-url>
cd tallyport
```

### 2. Install Dependencies
Ensure you have Go installed. Install the required Go modules:
```bash
go mod tidy
```
This will download dependencies like `github.com/go-chi/chi/v5`, `github.com/prometheus/client_golang`, and `github.com/rs/zerolog`.

### 3. Configure the Server
Create a `setting.yml` file in the project root with the following structure:
```yaml
ServerConfig:
  MaxHeaderBytes: 1048576 # 1MB
  ReadHeaderTimeout: 5000000000 # 5 seconds (in nanoseconds)
  WriteTimeout: 10000000000 # 10 seconds
  ReadTimeout: 10000000000 # 10 seconds
  IdleTimeout: 120000000000 # 120 seconds
  ServerName: "tallyport"
  Port: ":8080"
  TlsPath: "" # Set to path for TLS certs if needed
```
Adjust the values as needed for your environment.

### 4. Build and Run the Server
Build and run the Go server:
```bash
go build -o tallyport
./tallyport -config-file setting.yml
```
The server will start on `http://localhost:8080` (or the port specified in `setting.yml`).

### 5. Configure Prometheus
Update your Prometheus configuration (`prometheus.yml`) to scrape metrics from TallyPort:
```yaml
scrape_configs:
  - job_name: 'tallyport'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
```
Restart Prometheus to apply the changes:
```bash
prometheus --config.file=prometheus.yml
```

## Using TallyPort from a React Native Application

### 1. Set Up a React Native Project
If you don’t have a React Native project, create one:
```bash
npx react-native init MetricsApp
cd MetricsApp
```
Install the `axios` library for HTTP requests:
```bash
npm install axios
```

### 2. Initialize a Metric
To create a new metric, send a POST request to the `/init` endpoint. Below is an example of initializing a counter metric from React Native.

**Example: Initialize a Counter**
```javascript
import axios from 'axios';

const TALLYPORT_URL = 'http://localhost:8080'; // Replace with your server URL

async function initializeCounter() {
  try {
    const response = await axios.post(`${TALLYPORT_URL}/init`, {
      type: 'counter',
      name: 'app_button_clicks_total',
      description: 'Total number of button clicks in the app',
      labels: ['screen', 'button'],
    }, {
      headers: { 'Content-Type': 'application/json' },
    });
    console.log(response.data.message);
  } catch (error) {
    console.error('Error initializing counter:', error.response?.data || error.message);
  }
}
```

### 3. Push Metric Updates
To update a metric, send a POST request to the `/push` endpoint with the metric type, name, labels, and values.

**Example: Push a Counter Update**
```javascript
async function pushCounterUpdate() {
  try {
    const response = await axios.post(`${TALLYPORT_URL}/push`, {
      type: 'counter',
      name: 'app_button_clicks_total',
      labels: ['home', 'submit'],
    }, {
      headers: { 'Content-Type': 'application/json' },
    });
    console.log(response.data.message);
  } catch (error) {
    console.error('Error pushing counter:', error.response?.data || error.message);
  }
}
```

**Example: Push a Gauge Update**
```javascript
async function pushGaugeUpdate() {
  try {
    const response = await axios.post(`${TALLYPORT_URL}/push`, {
      type: 'gauge',
      name: 'app_cpu_usage',
      labels: ['device'],
      gauge: {
        label_values: ['mobile'],
        values: [75.5],
      },
    }, {
      headers: { 'Content-Type': 'application/json' },
    });
    console.log(response.data.message);
  } catch (error) {
    console.error('Error pushing gauge:', error.response?.data || error.message);
  }
}
```

**Example: Push a Histogram Update**
```javascript
async function pushHistogramUpdate() {
  try {
    const response = await axios.post(`${TALLYPORT_URL}/push`, {
      type: 'histogram',
      name: 'app_request_duration_seconds',
      labels: ['endpoint'],
      histogram: {
        observations: [{ label: 'api_call', value: 0.123 }],
      },
    }, {
      headers: { 'Content-Type': 'application/json' },
    });
    console.log(response.data.message);
  } catch (error) {
    console.error('Error pushing histogram:', error.response?.data || error.message);
  }
}
```

### 4. Example React Native Component
Below is a simple React Native component that initializes and updates a counter metric when a button is pressed:

```javascript
import React from 'react';
import { View, Button, Text } from 'react-native';
import axios from 'axios';

const TALLYPORT_URL = 'http://localhost:8080';

const App = () => {
  const initializeMetric = async () => {
    try {
      const response = await axios.post(`${TALLYPORT_URL}/init`, {
        type: 'counter',
        name: 'app_button_clicks_total',
        description: 'Total number of button clicks in the app',
        labels: ['screen', 'button'],
      }, {
        headers: { 'Content-Type': 'application/json' },
      });
      console.log(response.data.message);
    } catch (error) {
      console.error('Error initializing metric:', error.response?.data || error.message);
    }
  };

  const pushMetric = async () => {
    try {
      const response = await axios.post(`${TALLYPORT_URL}/push`, {
        type: 'counter',
        name: 'app_button_clicks_total',
        labels: ['home', 'submit'],
      }, {
        headers: { 'Content-Type': 'application/json' },
      });
      console.log(response.data.message);
    } catch (error) {
      console.error('Error pushing metric:', error.response?.data || error.message);
    }
  };

  return (
    <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
      <Button title="Initialize Metric" onPress={initializeMetric} />
      <Button title="Record Button Click" onPress={pushMetric} />
      <Text>Press to send metrics to TallyPort</Text>
    </View>
  );
};

export default App;
```

### 5. Run the React Native App
Start the React Native app:
```bash
npx react-native run-android
# or
npx react-native run-ios
```
Ensure the TallyPort server is running and accessible from the React Native app (e.g., use the correct IP address if running on a physical device).

## API Endpoints

### `/init`
**Method**: POST  
**Content-Type**: `application/json`  
**Purpose**: Creates a new Prometheus metric.  
**Request Body**:
```json
{
  "type": "counter|gauge|histogram|summary",
  "name": "metric_name",
  "description": "Metric description",
  "labels": ["label1", "label2"],
  "histogram": {
    "buckets": [0.1, 0.5, 1.0] // For histogram only
  },
  "summary": {
    "objectives": { "0.5": 0.05, "0.9": 0.01 }, // For summary only
    "max_age": 3600000000000 // For summary, in nanoseconds
  }
}
```
**Response**:
```json
{ "message": "Metric metric_name created successfully" }
```

### `/push`
**Method**: POST  
**Content-Type**: `application/json`  
**Purpose**: Updates an existing metric with new values.  
**Request Body**:
```json
{
  "type": "counter|gauge|histogram|summary",
  "name": "metric_name",
  "labels": ["value1", "value2"],
  "gauge": {
    "label_values": ["value1"],
    "values": [42.0]
  },
  "histogram": {
    "observations": [{ "label": "value1", "value": 0.123 }]
  }
}
```
**Response**:
```json
{ "message": "Metric metric_name updated" }
```

### `/metrics`
**Method**: GET  
**Purpose**: Exposes Prometheus metrics for scraping.  
**Response**: Prometheus text format.

## Troubleshooting
- **Metric Not Found**: Ensure the metric was initialized via `/init` before pushing updates.
- **Invalid JSON**: Check the request body format and ensure `Content-Type: application/json`.
- **Prometheus Not Scraping**: Verify the Prometheus configuration and ensure the server is reachable.
- **CORS Issues**: If accessing from a React Native app on a different network, add CORS middleware to the server:
  ```go
  r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}}))
  ```
  And import `github.com/go-chi/cors`.

## License
MIT License. See `LICENSE` file for details.

