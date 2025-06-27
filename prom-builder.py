from os import path,getcwd,environ
from yaml import safe_dump,safe_load,YAMLError
import sys
import re
import json
from typing import Any, List, Union
from dotenv import load_dotenv
import glob

def load_population_file(file_path: str) -> Any:
    """Load and parse a JSON file (e.g., populate.json)."""
    try:
        with open(file_path, 'r') as file:
            return json.load(file)
    except FileNotFoundError:
        print(f"Error: File {file_path} not found.")
        raise
    except json.JSONDecodeError as e:
        print(f"Error parsing JSON file: {e}")
        raise

def load_yaml_file(file_path: str) -> Any:
    """Load and parse a YAML file (e.g., prometheus.yml)."""
    try:
        with open(file_path, 'r') as file:
            return safe_load(file) or {} 
    except FileNotFoundError:
        print(f"Error: File {file_path} not found.")
        raise
    except YAMLError as e:
        print(f"Error parsing YAML file: {e}")
        raise

def merge_configs(config1: Any, config2: Any) -> Any:
    """Merge two YAML configurations, handling nested dictionaries and lists."""
    if isinstance(config1, dict) and isinstance(config2, dict):
        merged = dict(config1)
        for key, value in config2.items():
            if key in merged and isinstance(merged[key], (dict, list)):
                merged[key] = merge_configs(merged[key], value)
            else:
                merged[key] = value
        return merged
    elif isinstance(config1, list) and isinstance(config2, list):
        return config1 + config2  
    return config2 if config2 is not None else config1


def validate_config(data: dict, expected: List[Union[int, str]]) -> bool:
    if not isinstance(data, dict):
        return False
    for key in expected:
        if not isinstance(key, (int, str)):
            raise TypeError(f"Expected key {key} must be an integer or string")
    return all(key in data for key in expected)

def process_population_path(data: Any) -> list:
    """Process the population file and load all YAML files specified in paths."""
    yaml_configs = []
    expected = ["skip_merge","path"]
    for config in data:
        if not validate_config(config,expected):
            return yaml_configs
        if config["skip_merge"]:
            continue
        if 'path' not in config:
            raise Exception(f"No path found in configuration: {config}")
        path = config['path']
        # Handle glob patterns (e.g., ./alerts/**.yml)
        if '**' in path or '*' in path:
            matching_files = glob.glob(path, recursive=True)
            if not matching_files:
                print(f"Warning: No files found for pattern {path}")
            for file_path in matching_files:
                yaml_configs.append(load_yaml_file(file_path))
        else:
            yaml_configs.append(load_yaml_file(path))
    return yaml_configs

def resolve_env_vars(data: Any, env_vars: dict) -> Any:
    """Recursively replace environment variable placeholders in the YAML data."""
    if isinstance(data, dict):
        return {k: resolve_env_vars(v, env_vars) for k, v in data.items()}
    elif isinstance(data, list):
        return [resolve_env_vars(item, env_vars) for item in data]
    elif isinstance(data, str):
        data = data.strip()
        def replace_match(match):
            var_name = match.group(1)
            default_value = match.group(2)[1:] if match.group(2) else None
            return env_vars.get(var_name, default_value or match.group(0))
        return re.sub(r'\${(\w+)(?::([^}]*))?}', replace_match, data)
    return data

def save_yaml_file(file_path: str, data: Any) -> None:
    """Save the processed YAML data back to the specified file."""
    try:
        with open(file_path, 'w') as file:
            safe_dump(data, file, default_flow_style=False, allow_unicode=True)
    except IOError as e:
        print(f"Error writing to file {file_path}: {e}")
        raise

def main() -> None:
    """Process YAML files by merging configurations and resolving environment variables."""
    if len(sys.argv) < 2:
        print("Error: YAML file and population file must be specified." \
        " Usage: python resolve_env_vars.py <output_yaml_file> <population_json_file>")
        sys.exit(1)

    if path.exists(getcwd().join(".env")):
        load_dotenv(getcwd().join(".env"))
        
    yaml_file = sys.argv[1] 
    population_path = sys.argv[2]
    env_vars = dict(environ)
    full_config = {}
    if population_path == "-" :
        try:
            base_config = load_yaml_file(yaml_file)
            resolved_config = resolve_env_vars(base_config, env_vars)
            full_config = merge_configs(full_config, resolved_config)
            save_yaml_file(yaml_file, full_config)
        except FileNotFoundError:
            print(f"Warning: Base YAML file {yaml_file} not found.")
        return
    
    population_data = load_population_file(population_path)
    partial_configs = process_population_path(population_data)
    try:
        base_config = load_yaml_file(yaml_file)
        partial_configs.append(base_config)
    except FileNotFoundError:
        print(f"Warning: Base YAML file {yaml_file} not found. Proceeding with population configs only.")
        base_config = {}

    for config in partial_configs:
        full_config = merge_configs(full_config, config)
    resolved_config = resolve_env_vars(full_config, env_vars)
    save_yaml_file(yaml_file, resolved_config)
    print(f"Successfully processed {yaml_file} with merged configurations and environment variables.")

if __name__ == "__main__":
    main()