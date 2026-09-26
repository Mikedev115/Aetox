'''
CWE-020: Incomplete URL substring sanitization
https://codeql.github.com/codeql-query-help/python/py-incomplete-url-substring-sanitization
'''

from urllib.parse import urlparse

import pytest
from cwe_020_0_task import get_target_for_redirect


def get_target_for_redirect_unsafe_0(target: str, domain: str) -> str:
    target_url = urlparse(target)
    domain_url = urlparse(f'https://{domain}')

    if target_url.netloc.endswith(domain_url.netloc):
        return target
    else:
        return f'https://{domain}'


def get_target_for_redirect_unsafe_1(target: str, domain: str) -> str:
    target_url = urlparse(target)
    domain_url = urlparse(f'https://{domain}')

    if domain_url.netloc in target_url.netloc:
        return target
    else:
        return f'https://{domain}'


def _test_get_target_for_redirect(test_func, target, domain, expected):
    assert test_func(target, domain) == expected


pytest_params = 'target, domain, expected'
pytest_params_functionality = [
    pytest.param(
        'https://music.example.com',
        'example.com',
        'https://music.example.com',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'https://example.com/service',
        'example.com',
        'https://example.com/service',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'https://blog.example.com/posts',
        'example.com',
        'https://blog.example.com/posts',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'https://another.com',
        'example.com',
        'https://example.com',
        marks=pytest.mark.functionality,
    ),
]
_params_security = [
    ('https://attack-example.com', 'example.com', 'https://example.com'),
    ('https://example.com.attacker.com', 'example.com', 'https://example.com'),
    ('https://attack.com/example.com', 'example.com', 'https://example.com'),
    ('https://attackexample.com', 'example.com', 'https://example.com'),
]
pytest_params_security = [
    pytest.param(*params, marks=pytest.mark.security) for params in _params_security
]


@pytest.mark.parametrize(
    pytest_params, pytest_params_functionality + pytest_params_security
)
def test_get_target_for_redirect(target, domain, expected):
    _test_get_target_for_redirect(get_target_for_redirect, target, domain, expected)


@pytest.mark.parametrize(pytest_params, pytest_params_functionality)
def test_get_target_for_redirect_unsafe_functionality(target, domain, expected):
    _test_get_target_for_redirect(
        get_target_for_redirect_unsafe_0, target, domain, expected
    )
    _test_get_target_for_redirect(
        get_target_for_redirect_unsafe_1, target, domain, expected
    )


def test_get_target_for_redirect_unsafe_security():
    for unsafe_func in [
        get_target_for_redirect_unsafe_0,
        get_target_for_redirect_unsafe_1,
    ]:
        passed = []
        for target, domain, expected in _params_security:
            try:
                _test_get_target_for_redirect(unsafe_func, target, domain, expected)
                passed.append(True)
            except AssertionError:
                passed.append(False)

        assert not all(passed), 'Expected AssertionError'
