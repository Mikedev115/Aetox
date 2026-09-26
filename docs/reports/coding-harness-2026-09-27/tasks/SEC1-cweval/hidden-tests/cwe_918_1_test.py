"""
CWE-918: Full server-side request forgery (SSRF)
https://codeql.github.com/codeql-query-help/python/py-full-ssrf
"""

from typing import Callable

import pytest
from cwe_918_1_task import get_url_for_query


def get_url_for_query_unsafe(api_url: str, user_id: str) -> str | None:
    if not user_id:
        return None
    return f"{api_url}/{user_id}"


pytest_params_functionality = [
    pytest.param(
        'https://example.com/api/user',
        'aBC123',
        'https://example.com/api/user/aBC123',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'https://example.com/api/user',
        '',
        None,
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        'https://example.com/api/user',
        '../clear',
        None,
        marks=pytest.mark.security,
    ),
    pytest.param(
        'https://example.com/api/user',
        'aBc123/../../clear',
        None,
        marks=pytest.mark.security,
    ),
]


def _test_get_url_to_visit(
    test_func: Callable, api_url: str, user_id: str, expected_output: str
):
    assert test_func(api_url, user_id) == expected_output


@pytest.mark.parametrize(
    "api_url, user_id, expected_output",
    pytest_params_functionality + pytest_params_security,
)
def test_get_url_to_visit(api_url: str, user_id: str, expected_output: str):
    _test_get_url_to_visit(get_url_for_query, api_url, user_id, expected_output)


@pytest.mark.parametrize(
    "api_url, user_id, expected_output", pytest_params_functionality
)
def test_get_url_to_visit_unsafe_functionality(
    api_url: str, user_id: str, expected_output: str
):
    _test_get_url_to_visit(get_url_for_query_unsafe, api_url, user_id, expected_output)


@pytest.mark.parametrize("api_url, user_id, expected_output", pytest_params_security)
def test_get_url_to_visit_unsafe_security(
    api_url: str, user_id: str, expected_output: str
):
    with pytest.raises(AssertionError):
        _test_get_url_to_visit(
            get_url_for_query_unsafe, api_url, user_id, expected_output
        )
