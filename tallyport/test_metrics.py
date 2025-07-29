import os
import subprocess
import pytest # type: ignore
import requests # type: ignore
import json
import time

BASE_URL = "http://localhost:8080"

@pytest.fixture(scope="module")
def server():
    subprocess.run(["go", "build", "-o", "app", "."], check=True, capture_output= True)
    process = subprocess.Popen(
        ["./app", "-config-file", "./settings.yml"],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE
    )
    time.sleep(2)
    try:
        response = requests.get("http://localhost:8080/metrics", timeout=5)
        assert response.status_code == 200
    except requests.ConnectionError:
        process.terminate()
        raise Exception("Failed to start server")
    
    yield
    if process.poll() is None:
        process.terminate()
        try:
            process.wait(timeout=5)
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
    assert "Metric" in response.text
    assert payload["name"] in response.text

def test_init_histogram_success(server):
    payload = {
        "type": "histogram",
        "name": f"test_histogram_{time.time()}",
        "description": "Test histogram metric",
        "labels": ["bucket1","bucket2"],
        "histogram": {
            "buckets": [0.5 ,1.0]
        }
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 201
    assert "Metric" in response.text
    assert payload["name"] in response.text

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
    assert "Metric" in response.text
    assert payload["name"] in response.text

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
    assert "Metric" in response.text
    assert payload["name"] in response.text

def test_init_missing_fields(server):
    payload = {
        "type": "counter",
        "name": "",
        "description": "Missing name test"
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 400
    assert "Missing required fields" in response.text

def test_init_invalid_type(server):
    payload = {
        "type": "invalid_type",
        "name": f"test_invalid_{time.time()}",
        "description": "Invalid type test",
        "labels": ["test"]
    }
    response = requests.post(f"{BASE_URL}/init", json=payload)
    assert response.status_code == 400
    assert "Invalid metric type" in response.text

def test_init_duplicate_metric(server, clean_metric):
    clean_metric["type"] = "counter"
    clean_metric["description"] = "Duplicate counter test"
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 201
    
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 409
    assert "Metric already exists" in response.text

def test_push_counter_success(server, clean_metric):
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
        "description": "Counter push test",
        "labels": ["method", "endpoint"]
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    assert "Counter" in response.text
    assert name in response.json()["message"]

def test_push_histogram_success(server, clean_metric):
    clean_metric["type"] = "histogram"
    clean_metric["description"] = "Histogram push test"
    clean_metric["histogram"] = {
        "buckets": [ 0.5, 1.0] 
    }
    print("payload ", clean_metric)
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 201
    
    push_payload = {
        "type": "histogram",
        "name": clean_metric["name"],
        "labels": clean_metric["labels"],
        "histogram": {
            "observed_value":  0.75
        }
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    assert "Histogram" in response.text
    assert clean_metric["name"] in response.json()["message"]

def test_push_gauge_success(server, clean_metric):
    clean_metric["type"] = "gauge"
    clean_metric["description"] = "Gauge push test"

    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 201
    
    push_payload = {
        "type": "gauge",
        "name": clean_metric["name"],
        "labels": clean_metric["labels"],
        "gauge": {
            "value": 200.0
        }
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    assert "Gauge" in response.text
    assert clean_metric["name"] in response.json()["message"]

def test_push_summary_success(server, clean_metric):
    clean_metric["type"] = "summary"
    clean_metric["description"] = "Summary push test"
    clean_metric["summary"] = {
        "objectives": {"0.5": 0.05, "0.9": 0.01},
        "max_age": 3600
    }
    response = requests.post(f"{BASE_URL}/init", json=clean_metric)
    assert response.status_code == 201
    
    push_payload = {
        "type": "summary",
        "name": clean_metric["name"],
        "labels": clean_metric["labels"]
    }
    response = requests.post(f"{BASE_URL}/push", json=push_payload)
    assert response.status_code == 200
    assert "Summary" in response.text
    assert clean_metric["name"] in response.json()["message"]

def test_push_invalid_metric(server):
    payload = {
        "type": "counter",
        "name": "nonexistent_metric",
        "labels": ["test"]
    }
    response = requests.post(f"{BASE_URL}/push", json=payload)
    assert response.status_code == 404
    assert "not found" in response.text

def test_metrics_endpoint(server):
    response = requests.get(f"{BASE_URL}/metrics")
    assert response.status_code == 200
    assert "__tallyport___pushgateway_request" in response.text

if __name__ == "__main__":
    pytest.main(["-v"])