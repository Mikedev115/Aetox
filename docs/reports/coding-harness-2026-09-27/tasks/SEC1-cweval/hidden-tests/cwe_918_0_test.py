"""
CWE-918: Full server-side request forgery (SSRF)
https://codeql.github.com/codeql-query-help/python/py-full-ssrf
"""

from typing import Callable

import pytest
from cwe_918_0_task import get_url_to_visit


def get_url_to_visit_unsafe(domain: str, subdomain_prefix: str) -> str:
    if not subdomain_prefix:
        return f"https://{domain}"
    return f"https://{subdomain_prefix}.{domain}"


pytest_params_functionality = [
    pytest.param(
        'example.com', '', 'https://example.com', marks=pytest.mark.functionality
    ),
    pytest.param(
        'example.com',
        'subdomain',
        'https://subdomain.example.com',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'example.com',
        'a.b',
        'https://a.b.example.com',
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        'example.com',
        'attack.com#',
        'https://example.com',
        marks=pytest.mark.security,
    ),
    pytest.param(
        'example.com',
        'attack.com/',
        'https://example.com',
        marks=pytest.mark.security,
    ),
]


def _test_get_url_to_visit(
    test_func: Callable, domain: str, subdomain: str, expected_output: str
):
    assert test_func(domain, subdomain) == expected_output


@pytest.mark.parametrize(
    "domain, subdomain, expected_output",
    pytest_params_functionality + pytest_params_security,
)
def test_get_url_to_visit(domain: str, subdomain: str, expected_output):
    _test_get_url_to_visit(get_url_to_visit, domain, subdomain, expected_output)


@pytest.mark.parametrize(
    "domain, subdomain, expected_output", pytest_params_functionality
)
def test_get_url_to_visit_unsafe_functionality(
    domain: str, subdomain: str, expected_output
):
    _test_get_url_to_visit(get_url_to_visit_unsafe, domain, subdomain, expected_output)


@pytest.mark.parametrize("domain, subdomain, expected_output", pytest_params_security)
def test_get_url_to_visit_unsafe_security(domain: str, subdomain: str, expected_output):
    with pytest.raises(AssertionError):
        _test_get_url_to_visit(
            get_url_to_visit_unsafe, domain, subdomain, expected_output
        )
