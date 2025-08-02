import os
import subprocess
import pytest # type: ignore
import requests
import json
import time

BASE_URL = "http://localhost:8080"

@pytest.fixture(scope="module")
def server():
    subprocess.run(["go", "build", "-o", "app", "."], check=True, capture_output=True)
    subprocess.run(["chmod", "+x", "./app"], check=True, capture_output=True)
    process = subprocess.Popen(
        ["./app", "-config-file", "./settings.yml"],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    time.sleep(10)
    try:
        response = requests.get(f"{BASE_URL}/metrics", timeout=10)
        assert response.status_code == 200
    except requests.ConnectionError:
        process.terminate()
        raise Exception("Failed to start server")
    
    yield
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
    if os.path.exists("app"):
        os.remove("app")

@pytest.fixture
def clean_metric():
    return {
        "type": "",
        "name": f"test_metric_{time.time()}",
        "description": "Test metric",
        "labels": ["bucket1", "bucket2"]
    }

def test_init_counter_success(server):
    payload = {
        "type": "counter",
        "name": f"test_counter_{time.time()}",
        "description": "Test counter metric",
        "labels": ["method", "endpoint"]
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["status"] == 201
    assert data["message"] == f"Metric {payload['name']} created successfully"

def test_init_histogram_success(server):
    payload = {
        "type": "histogram",
        "name": f"test_histogram_{time.time()}",
        "description": "Test histogram metric",
        "labels": ["bucket1", "bucket2"],
        "histogram": {
            "buckets": [0.5, 1.0]
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["status"] == 201
    assert data["message"] == f"Metric {payload['name']} created successfully"

def test_init_gauge_success(server):
    payload = {
        "type": "gauge",
        "name": f"test_gauge_{time.time()}",
        "description": "Test gauge metric",
        "labels": ["status"],
        "gauge": {
            "value": 100.0
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["status"] == 201
    assert data["message"] == f"Metric {payload['name']} created successfully"

def test_init_summary_success(server):
    payload = {
        "type": "summary",
        "name": f"test_summary_{time.time()}",
        "description": "Test summary metric",
        "labels": ["percentile"],
        "summary": {
            "objectives": {"0.5": 0.05, "0.9": 0.01},
            "max_age": 3600
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    data = response.json()
    assert data["status"] == 201
    assert data["message"] == f"Metric {payload['name']} created successfully"

def test_init_missing_fields(server):
    payload = {
        "type": "counter",
        "name": "",
        "description": "Missing name test"
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 400
    data = response.json()
    assert data["status"] == 400
    assert "field cannot be empty" in data["reason"]


def test_init_invalid_type(server):
    payload = {
        "type": "invalid_type",
        "name": f"test_invalid_{time.time()}",
        "description": "Invalid type test",
        "labels": ["test"]
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 400
    data = response.json()
    assert data["status"] == 400
    assert "value provided to field not supported" in data["reason"]
   

def test_init_duplicate_metric(server, clean_metric):
    clean_metric["type"] = "counter"
    clean_metric["description"] = "Duplicate counter test"
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 201
    data = response.json()
    assert data["status"] == 201
    assert data["message"] == f"Metric {clean_metric['name']} created successfully"
    
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 400
    data = response.json()
    assert data["status"] == 400
    assert "resource conflict" in data["reason"].lower()
    

def test_push_counter_success(server):
    name = f"test_counter_{time.time()}"
    payload = {
        "type": "counter",
        "name": name,
        "description": "Counter push test",
        "labels": ["method", "endpoint"]
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    
    push_payload = {
        "type": "counter",
        "name": name,
        "labels": ["method", "endpoint"]
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == 200
    assert data["message"] == f"Metric {name} updated successfully"

def test_push_histogram_success(server):
    name = f"test_histogram_{time.time()}"
    payload = {
        "type": "histogram",
        "name": name,
        "description": "Histogram push test",
        "labels": ["bucket1", "bucket2"],
        "histogram": {
            "buckets": [0.5, 1.0]
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    
    push_payload = {
        "type": "histogram",
        "name": name,
        "labels": ["bucket1", "bucket2"],
        "histogram": {
            "observed_value": 0.75
        }
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == 200
    assert data["message"] == f"Metric {name} updated successfully"

def test_push_gauge_success(server):
    name = f"test_gauge_{time.time()}"
    payload = {
        "type": "gauge",
        "name": name,
        "description": "Gauge push test",
        "labels": ["status"]
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    
    push_payload = {
        "type": "gauge",
        "name": name,
        "labels": ["status"],
        "gauge": {
            "value": 200.0
        }
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == 200
    assert data["message"] == f"Metric {name} updated successfully"

def test_push_summary_success(server):
    name = f"test_summary_{time.time()}"
    payload = {
        "type": "summary",
        "name": name,
        "description": "Summary push test",
        "labels": ["percentile"],
        "summary": {
            "objectives": {"0.5": 0.5, "0.9": 0.01},
            "max_age": 3600
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    
    push_payload = {
        "type": "summary",
        "name": name,
        "labels": ["percentile"]
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    data = response.json()
    assert data["status"] == 200
    assert data["message"] == f"Metric {name} updated successfully"

def test_push_invalid_metric(server, clean_metric):
    payload = {
        "type": "counter",
        "name": f"nonexistent_metric_{time.time()}",
        "labels": ["test"]
    }
    response = requests.post(f"{BASE_URL}/push", json=payload)
    assert response.status_code == 400
    data = response.json()
    assert data["status"] == 400
    assert "not found" in data["reason"].lower()

def test_metrics_endpoint(server):
    response = requests.get(f"{BASE_URL}/metrics")
    assert response.status_code == 200
    assert "__tallyport__" in response.text

if __name__ == "__main__":
    pytest.main(["-v"])