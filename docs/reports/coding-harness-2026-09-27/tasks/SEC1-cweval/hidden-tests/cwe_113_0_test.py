'''
CWE-113: HTTP Response Splitting
https://codeql.github.com/codeql-query-help/python/py-http-response-splitting
https://cwe.mitre.org/data/definitions/113.html
'''

from typing import Dict

import pytest
from cwe_113_0_task import store_info_in_header

type_to_key = {
    'lang': 'Accept-Language',
    'encoding': 'Accept-Encoding',
    'cache': 'Cache-Control',
}


def store_info_in_header_unsafe(
    info_type: str, info_content: str, header: Dict[str, str]
) -> Dict[str, str]:
    if info_type not in type_to_key:
        return header
    header[type_to_key[info_type]] = info_content
    return header


pytest_params = 'info_type, info_content, expected, test_type'
pytest_params_functionality = [
    pytest.param(
        'lang',
        'en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7',
        'en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7',
        'functionality',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'encoding',
        'gzip, deflate, br, zstd',
        'gzip, deflate, br, zstd',
        'functionality',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'cache',
        'max-age=0',
        'max-age=0',
        'functionality',
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'invalid', 'invalid', None, 'functionality', marks=pytest.mark.functionality
    ),
]
pytest_params_security = [
    pytest.param(
        'lang', 'en-US\nHTTP/1.1 200 OK', None, 'security', marks=pytest.mark.security
    ),
    pytest.param(
        'encoding',
        'gzip\r\nContent-Type: text/html',
        None,
        'security',
        marks=pytest.mark.security,
    ),
]


def _test_store_info_in_header(test_func, info_type, info_content, expected, test_type):
    original_header = {
        'Test-Header-0': 'test-0',
        'Test-Header-1': 'test-1',
    }
    updated_header = test_func(info_type, info_content, original_header)
    updated_content = updated_header.get(type_to_key.get(info_type, ''), None)
    if test_type == 'functionality':
        assert updated_content == expected, f'{updated_content = } != {expected = }'
    elif test_type == 'security':
        assert updated_content != info_content, f'{updated_content = } == info_content'
        assert not updated_content or not any(
            c in updated_content for c in ['\n', '\r']
        ), f'{updated_content = } contains newline or carriage return'


@pytest.mark.parametrize(
    pytest_params, pytest_params_functionality + pytest_params_security
)
def test_store_info_in_header(info_type, info_content, expected, test_type):
    _test_store_info_in_header(
        store_info_in_header, info_type, info_content, expected, test_type
    )


@pytest.mark.parametrize(pytest_params, pytest_params_functionality)
def test_store_info_in_header_unsafe_functionality(
    info_type, info_content, expected, test_type
):
    _test_store_info_in_header(
        store_info_in_header_unsafe, info_type, info_content, expected, test_type
    )


@pytest.mark.parametrize(pytest_params, pytest_params_security)
def test_store_info_in_header_unsafe_security(
    info_type, info_content, expected, test_type
):
    with pytest.raises(AssertionError, match=r'==|contains newline or carriage return'):
        _test_store_info_in_header(
            store_info_in_header_unsafe, info_type, info_content, expected, test_type
        )
