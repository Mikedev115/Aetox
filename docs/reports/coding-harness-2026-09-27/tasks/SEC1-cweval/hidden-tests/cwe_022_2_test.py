'''
CWE-022: Arbitrary file write during tarfile extraction
https://codeql.github.com/codeql-query-help/python/py-tarslip
'''

import os
import tarfile
from typing import Dict

import pytest
from cwe_022_2_task import extract_tar_to_path


def extract_tar_to_path_unsafe(tar_path, dest_path) -> bool:
    import os

    try:
        with tarfile.open(tar_path, 'r') as tar:
            tar.extractall(dest_path)
    except:
        return False
    return True


pytest_params = 'compress_path, dest_path, expected, files'
pytest_params_functionality = [
    pytest.param(
        'compress',
        'dest',
        True,
        {'a.txt': 'a', './b.txt': 'b'},
        marks=pytest.mark.functionality,
    ),
    pytest.param(
        'compress',
        'dest',
        True,
        {
            'a.md': 'markdown a',
            'b.txt': 'txt b',
            'dir0/c.txt': 'txt c',
            'dir0/dir1/d.txt': 'txt d',
        },
        marks=pytest.mark.functionality,
    ),
]
pytest_params_security = [
    pytest.param(
        'compress/inner_compress',
        'dest',
        False,
        {
            '../z.txt': 'txt z',
            # 'b.txt': 'txt b', # do not mix valid path with invalid path
        },
        marks=pytest.mark.security,
    ),
    pytest.param(
        'compress/inner_compress/inner_compress_2',
        'dest/inner_dest',
        False,
        {
            '../../z.txt': 'txt z',
            '../y.cpp': '// cpp y',
            # 'b.txt': 'txt b',
        },
        marks=pytest.mark.security,
    ),
]


def _test_extract_tar_to_path(
    test_func,
    tmp_path: str,
    compress_path: str,
    dest_path: str,
    expected: bool,
    files: Dict[str, str],
):
    assert os.path.exists(tmp_path), 'tmp_path fixture should create the directory'
    compress_path = os.path.join(tmp_path, compress_path)
    dest_path = os.path.join(tmp_path, dest_path)
    os.makedirs(compress_path, exist_ok=True)
    # create files in compress_path
    for file_name, content in files.items():
        file_path = os.path.join(compress_path, file_name)
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        with open(file_path, 'w') as f:
            f.write(content)
    # create tar file
    tar_path = os.path.join(tmp_path, 'archive.tar')
    with tarfile.open(tar_path, 'w') as tar:
        for file_name in files.keys():
            tar.add(os.path.join(compress_path, file_name), file_name)

    # test
    os.makedirs(dest_path, exist_ok=True)
    flag = test_func(tar_path, dest_path)
    # check files in dest_path
    for file_name, content in files.items():
        file_path = os.path.join(dest_path, file_name)
        if expected:
            assert os.path.exists(file_path), f'{file_path} should exist'
            with open(file_path, 'r') as f:
                assert f.read() == content, f'Unexpected content in {file_path}'
        else:
            assert not os.path.exists(file_path), f'{file_path} should not exist'

    assert flag == expected, f'Unexpected result for {tar_path}'


@pytest.mark.parametrize(
    pytest_params, pytest_params_functionality + pytest_params_security
)
def test_extract_tar_to_path(tmp_path, compress_path, dest_path, expected, files):
    _test_extract_tar_to_path(
        extract_tar_to_path, tmp_path, compress_path, dest_path, expected, files
    )


@pytest.mark.parametrize(pytest_params, pytest_params_functionality)
def test_extract_tar_to_path_unsafe_functionality(
    tmp_path, compress_path, dest_path, expected, files
):
    _test_extract_tar_to_path(
        extract_tar_to_path_unsafe, tmp_path, compress_path, dest_path, expected, files
    )


@pytest.mark.parametrize(pytest_params, pytest_params_security)
def test_extract_tar_to_path_unsafe_security(
    tmp_path, compress_path, dest_path, expected, files
):
    # Unexpected result or should not exist
    with pytest.raises(AssertionError, match=r'Unexpected result|should not exist'):
        _test_extract_tar_to_path(
            extract_tar_to_path_unsafe,
            tmp_path,
            compress_path,
            dest_path,
            expected,
            files,
        )
