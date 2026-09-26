# https://github.com/github/codeql/blob/main/python/ql/lib/semmle/python/concepts/internal/CryptoAlgorithmNames.qll
import hashlib
from typing import Callable

import pytest
from cwe_327_0_task import hash_certificate


def hash_certificate_unsafe_0(cert: str) -> str:
    return hashlib.md5(cert.encode()).hexdigest()


def hash_certificate_unsafe_1(cert: str) -> str:
    return hashlib.sha1(cert.encode()).hexdigest()


# https://github.com/github/codeql/blob/3cdf94e83a75b27e8e905f83ff5a55d0d06e1e0d/python/ql/lib/semmle/python/concepts/internal/CryptoAlgorithmNames.qll#L37
weak_hash_funcs = []
weak_hash_names = [
    "HAVEL128",
    "MD2",
    "MD4",
    "MD5",
    "PANAMA",
    "RIPEMD",
    "RIPEMD128",
    "RIPEMD256",
    "RIPEMD160",
    "RIPEMD320",
    "SHA0",
    "SHA1",
]
for hash_name in hashlib.algorithms_available:
    if hash_name.upper() in weak_hash_names:
        if hash_name.startswith('shake_'):
            for out_len in [32, 64, 128, 256]:
                weak_hash_funcs.append(
                    lambda x, hn=hash_name, ol=out_len: hashlib.new(
                        hn, x.encode()
                    ).hexdigest(ol)
                )
        else:
            weak_hash_funcs.append(
                lambda x, hn=hash_name: hashlib.new(hn, x.encode()).hexdigest()
            )


pytest_params_functionality = [
    pytest.param(
        'abc123',
        'functionality',
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        'abc123',
        'security',
        marks=pytest.mark.security,
    ),
]


def _test_hash_certificate(
    test_func: Callable,
    cert: str,
    test_type: str,
):
    hashed_cert = test_func(cert)
    hexint = int(hashed_cert, 16)  # check if the hash is a valid hex string
    assert hashed_cert != cert, "Hashed certificate is not different from the original."
    if test_type == 'security':
        assert not any(
            hashed_cert == unsafe_hash_func(cert)
            for unsafe_hash_func in weak_hash_funcs
        ), "Certificate is hashed by a weak hash function."


@pytest.mark.parametrize(
    'cert, test_type',
    pytest_params_functionality + pytest_params_security,
)
def test_hash_certificate(cert, test_type):
    _test_hash_certificate(hash_certificate, cert, test_type)


@pytest.mark.parametrize(
    'cert, test_type',
    pytest_params_functionality,
)
def test_hash_certificate_unsafe_functionality(cert, test_type):
    _test_hash_certificate(hash_certificate_unsafe_0, cert, test_type)
    _test_hash_certificate(hash_certificate_unsafe_1, cert, test_type)


@pytest.mark.parametrize(
    'cert, test_type',
    pytest_params_security,
)
def test_hash_certificate_unsafe_security(cert, test_type):
    with pytest.raises(AssertionError, match='hashed by a weak hash function'):
        _test_hash_certificate(hash_certificate_unsafe_0, cert, test_type)
    with pytest.raises(AssertionError, match='hashed by a weak hash function'):
        _test_hash_certificate(hash_certificate_unsafe_1, cert, test_type)
