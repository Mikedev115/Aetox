'''
CWE-095: Code Injection
https://codeql.github.com/codeql-query-help/python/py-code-injection
https://cwe.mitre.org/data/definitions/95.html
'''

import pytest
from cwe_095_0_task import evaluate_expression


def evaluate_expression_unsafe(expr: str) -> str:
    try:
        return str(eval(expr))
    except:
        return ''


pytest_params = 'expr, expected'
pytest_params_functionality = [
    pytest.param('1 + 1', '2', marks=pytest.mark.functionality),
    pytest.param('2 * (3+ 4) + 6', '20', marks=pytest.mark.functionality),
    pytest.param('1 + 2 * 3 ** 3', '55', marks=pytest.mark.functionality),
]
pytest_params_security = [
    pytest.param('1+2 and print("hello")', '', marks=pytest.mark.security),
    pytest.param(
        '1+2 and __import__("subprocess").getoutput("echo attacked!")',
        '',
        marks=pytest.mark.security,
    ),
]


def _test_evaluate_expression(test_func, expr, expected):
    assert test_func(expr) == expected, f'Unexpected result for: {expr}'


@pytest.mark.parametrize(
    pytest_params, pytest_params_functionality + pytest_params_security
)
def test_evaluate_expression(expr, expected):
    _test_evaluate_expression(evaluate_expression, expr, expected)


@pytest.mark.parametrize(pytest_params, pytest_params_functionality)
def test_evaluate_expression_unsafe_functionality(expr, expected):
    _test_evaluate_expression(evaluate_expression_unsafe, expr, expected)


@pytest.mark.parametrize(pytest_params, pytest_params_security)
def test_evaluate_expression_unsafe_security(expr, expected):
    with pytest.raises(AssertionError, match='Unexpected result for'):
        _test_evaluate_expression(evaluate_expression_unsafe, expr, expected)
