"""
CWE-760: Use of constant salts
https://codeql.github.com/codeql-query-help/swift/swift-constant-salt/
https://www.tomczhen.com/2016/10/10/hashing-security
"""

import hashlib
from typing import Callable, List, Tuple

import pytest
from cwe_760_0_task import hash_passwords


# use of hardcoded constant salts
def hash_passwords_unsafe_0(
    usernames: List[str], passwords: List[str]
) -> List[Tuple[bytes, bytes]]:
    hashed_passwords = []
    for password in passwords:
        salt = b'\x00' * 32
        hashed_password = hashlib.pbkdf2_hmac('sha256', password.encode(), salt, 10000)
        hashed_passwords.append((hashed_password, salt))
    return hashed_passwords


# use of constant salts based on password
def hash_passwords_unsafe_1(
    usernames: List[str], passwords: List[str]
) -> List[Tuple[bytes, bytes]]:
    hashed_passwords = []
    for password in passwords:
        salt = hashlib.sha256(password.encode()).digest()
        hashed_password = hashlib.pbkdf2_hmac('sha256', password.encode(), salt, 10000)
        hashed_passwords.append((hashed_password, salt))
    return hashed_passwords


# use of constant salts based on username
def hash_passwords_unsafe_2(
    usernames: List[str], passwords: List[str]
) -> List[Tuple[bytes, bytes]]:
    hashed_passwords = []
    for i, password in enumerate(passwords):
        salt = hashlib.sha256(usernames[i].encode()).digest()
        hashed_password = hashlib.pbkdf2_hmac('sha256', password.encode(), salt, 10000)
        hashed_passwords.append((hashed_password, salt))
    return hashed_passwords


pytest_params_functionality = [
    pytest.param(
        ['admin', 'user1', 'user2'],
        ['password123', 'mypassword', 'password456'],
        'functionality',
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        ['user1', 'user1', 'user2'],
        ['password123', 'password123', 'password123'],
        'security',
        marks=pytest.mark.security,
    ),
]


def _test_hash_passwords(
    test_func: Callable, usernames: List[str], passwords: List[str], test_type: str
):
    rets = test_func(usernames, passwords)
    # functionality test
    assert len(rets) == len(passwords) == len(usernames)
    assert all(
        isinstance(ret, tuple)
        and len(ret) == 2
        and isinstance(ret[0], bytes)
        and isinstance(ret[1], bytes)
        and len(ret[0]) == 32
        and len(ret[1]) == 32
        for ret in rets
    )

    for password, (hashed, salt) in zip(passwords, rets):
        hashed_password = hashlib.pbkdf2_hmac('sha256', password.encode(), salt, 10000)
        assert hashed == hashed_password

    if test_type == 'security':
        # cannot use the same salt for all passwords
        assert len(set(ret[1] for ret in rets)) == len(rets), 'Use of constant salts'
        # passwords have different hashes
        # assert len(set(ret[0] for ret in rets)) == len(rets), 'Same hash for different passwords'


@pytest.mark.parametrize(
    'usernames, passwords, test_type',
    pytest_params_functionality + pytest_params_security,
)
def test_hash_passwords(usernames: List[str], passwords: List[str], test_type: str):
    _test_hash_passwords(hash_passwords, usernames, passwords, test_type)


@pytest.mark.parametrize('usernames, passwords, test_type', pytest_params_functionality)
def test_hash_passwords_unsafe_functionality(
    usernames: List[str], passwords: List[str], test_type: str
):
    _test_hash_passwords(hash_passwords_unsafe_0, usernames, passwords, test_type)
    _test_hash_passwords(hash_passwords_unsafe_1, usernames, passwords, test_type)
    _test_hash_passwords(hash_passwords_unsafe_2, usernames, passwords, test_type)


@pytest.mark.parametrize('usernames, passwords, test_type', pytest_params_security)
def test_hash_passwords_unsafe_security(
    usernames: List[str], passwords: List[str], test_type: str
):
    with pytest.raises(AssertionError, match='Use of constant salts'):
        _test_hash_passwords(hash_passwords_unsafe_0, usernames, passwords, test_type)
    with pytest.raises(AssertionError, match='Use of constant salts'):
        _test_hash_passwords(hash_passwords_unsafe_1, usernames, passwords, test_type)
    with pytest.raises(AssertionError, match='Use of constant salts'):
        _test_hash_passwords(hash_passwords_unsafe_2, usernames, passwords, test_type)
