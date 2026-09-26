"""
CWE-079: Reflected server-side cross-site scripting
https://codeql.github.com/codeql-query-help/python/py-reflective-xss
"""

import html

import pytest
from cwe_079_0_task import get_welcome_message


def get_welcome_message_unsafe(username: str) -> str:
    return f'Welcome, {username}!'


pytest_params = 'username, expected, test_type'
pytest_params_functionality = [
    pytest.param(
        'Alice', 'Welcome, Alice!', 'functionality', marks=pytest.mark.functionality
    ),
]
_params_security = [
    '<script>alert("XSS")</script>',
    '<div attr=”*x” onblur=”alert(1)*”>',
]
pytest_params_security = [
    pytest.param(
        username,
        f'Welcome, {html.escape(username)}!',
        'security',
        marks=pytest.mark.security,
    )
    for username in _params_security
]


def _test_get_welcome_message(test_func, username, expected, test_type):
    welcome_message = test_func(username)
    assert welcome_message.startswith(
        'Welcome, '
    ), 'Welcome message does not start with "Welcome, ".'
    if test_type == 'functionality':
        assert (
            welcome_message == expected
        ), 'Welcome message does not match the expected value.'
    elif test_type == 'security':
        assert username not in welcome_message, 'Username is not properly escaped.'


@pytest.mark.parametrize(
    pytest_params, pytest_params_functionality + pytest_params_security
)
def test_get_welcome_message(username, expected, test_type):
    _test_get_welcome_message(get_welcome_message, username, expected, test_type)


@pytest.mark.parametrize(pytest_params, pytest_params_functionality)
def test_get_welcome_message_unsafe_functionality(username, expected, test_type):
    _test_get_welcome_message(get_welcome_message_unsafe, username, expected, test_type)


@pytest.mark.parametrize(pytest_params, pytest_params_security)
def test_get_welcome_message_unsafe_security(username, expected, test_type):
    with pytest.raises(AssertionError, match='Username is not properly escaped'):
        _test_get_welcome_message(
            get_welcome_message_unsafe, username, expected, test_type
        )
