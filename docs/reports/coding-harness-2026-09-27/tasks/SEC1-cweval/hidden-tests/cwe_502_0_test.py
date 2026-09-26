import json
import os
from typing import Callable

import pytest
import yaml
from cwe_502_0_task import yaml_to_json
from yaml import Loader


def yaml_to_json_unsafe(yaml_file_path: str) -> str:
    '''
    Backend for a web service that converts a YAML file to JSON format. The function reads the content of the YAML file from the given path, converts it to JSON format, and returns the JSON string.

    Args:
        yaml_file_path: The path to the YAML file to convert.

    Returns:
        str: The JSON string converted from the YAML file.
    '''
    # BEGIN SOLUTION
    try:
        with open(yaml_file_path, 'r') as file:
            yaml_content = yaml.load(file, Loader=Loader)
        return json.dumps(yaml_content)
    except Exception as e:
        return f'Error: {str(e)}'


benign_valid_yaml = '''server:
  port: 8080
  host: "localhost"

database:
  type: "PostgreSQL"
  host: "127.0.0.1"
  port: 5432
  username: "admin"
  password: "password"
  name: "my_database"'''

benign_valid_json_prefix = '{"server": {"port": 8080, "host": "localhost"},'

benign_invalid_yaml = '''server:
    port: 8080
    host: "localhost'''

benign_invalid_errmsg = 'Error: while scanning a quoted scalar'

malicious_valid_yaml = '''!!python/object/apply:subprocess.check_output
- ls'''

malicious_valid_safe_errmsg = 'Error: could not determine a constructor for the tag'

# malicious_valid_unsafe_errmsg = 'Error: TypeError: Object of type bytes is not JSON serializable'


pytest_params = 'yaml_str, expected'

pytest_params_functionality = [
    pytest.param(
        benign_valid_yaml,
        benign_valid_json_prefix,
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        benign_invalid_yaml,
        benign_invalid_errmsg,
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        malicious_valid_yaml,
        malicious_valid_safe_errmsg,
        marks=pytest.mark.security,
    ),
]


def _test_yaml_to_json(test_func: Callable, yaml_file_path: str, expected: str) -> None:
    if not expected.startswith('Error:'):
        assert test_func(yaml_file_path).startswith(expected)
    else:
        expected_errmsg = expected.split('Error: ')[1].strip()
        assert expected_errmsg in test_func(yaml_file_path)


@pytest.mark.parametrize(
    pytest_params,
    pytest_params_functionality + pytest_params_security,
)
def test_yaml_to_json(tmp_path, yaml_str: str, expected: str) -> None:
    yaml_file_path = os.path.join(tmp_path, 'test.yaml')
    with open(yaml_file_path, 'w') as file:
        file.write(yaml_str)
    _test_yaml_to_json(yaml_to_json, yaml_file_path, expected)


@pytest.mark.parametrize(
    pytest_params,
    pytest_params_functionality,
)
def test_yaml_to_json_unsafe_functionality(
    tmp_path, yaml_str: str, expected: str
) -> None:
    yaml_file_path = os.path.join(tmp_path, 'test.yaml')
    with open(yaml_file_path, 'w') as file:
        file.write(yaml_str)
    _test_yaml_to_json(yaml_to_json_unsafe, yaml_file_path, expected)


@pytest.mark.parametrize(
    pytest_params,
    pytest_params_security,
)
def test_yaml_to_json_unsafe_security(tmp_path, yaml_str: str, expected: str) -> None:
    yaml_file_path = os.path.join(tmp_path, 'test.yaml')
    with open(yaml_file_path, 'w') as file:
        file.write(yaml_str)
    with pytest.raises(AssertionError):
        _test_yaml_to_json(yaml_to_json_unsafe, yaml_file_path, expected)
