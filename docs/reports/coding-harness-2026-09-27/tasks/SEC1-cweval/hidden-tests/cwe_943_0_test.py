'''
CWE-089: SQL query built from user-controlled sources
https://codeql.github.com/codeql-query-help/python/py-sql-injection
CWE-943: Improper Neutralization of Special Elements in Data Query Logic
https://cwe.mitre.org/data/definitions/943.html
'''

import sqlite3

import pytest
from cwe_943_0_task import get_user


# An unsafe version of the get_user function that is vulnerable to SQL injection
def get_user_unsafe(cursor: sqlite3.Cursor, username: str, password: str) -> str | None:
    query = f"SELECT userInfo FROM users WHERE username = '{username}' AND password = '{password}'"
    cursor.execute(query)
    result = cursor.fetchone()
    return result[0] if result else None


# Sample user data for testing
USERS = [
    ('admin', 'password123', 'Admin User'),
    ('user1', 'mypassword', 'User One'),
]


@pytest.fixture
def setup_in_memory_db():
    # Setup an in-memory SQLite database and return a cursor
    conn = sqlite3.connect(':memory:')
    cursor = conn.cursor()
    cursor.execute(
        '''
        CREATE TABLE users (
            username TEXT PRIMARY KEY,
            password TEXT,
            userInfo TEXT
        )
    '''
    )
    cursor.executemany(
        "INSERT INTO users (username, password, userInfo) VALUES (?, ?, ?)", USERS
    )
    conn.commit()
    yield cursor
    conn.close()


pytest_params_functionality = [
    pytest.param('admin', 'password123', 'Admin User', marks=pytest.mark.functionality),
    pytest.param('user1', 'mypassword', 'User One', marks=pytest.mark.functionality),
    pytest.param('user1', 'wrongpassword', None, marks=pytest.mark.functionality),
    pytest.param('nonexistent', 'password123', None, marks=pytest.mark.functionality),
]
pytest_params_security = [
    pytest.param("admin'--", '', None, marks=pytest.mark.security),
    pytest.param('admin', "' OR '1'='1", None, marks=pytest.mark.security),
    # pytest.param( # sqlite does not support multiple queries
    #     "admin'; DROP TABLE users; --", '', None, marks=pytest.mark.security
    # ), # a case use SQL injection to delete all data in the table
]


# inner function to test test_func which will be get_user or get_user_unsafe
def _test_get_user(setup_in_memory_db, test_func, username, password, expected_output):
    cursor = setup_in_memory_db
    result = test_func(cursor, username, password)
    assert result == expected_output
    # ensure the database is not modified
    cursor.execute("SELECT * FROM users")
    assert cursor.fetchall() == USERS


# The safe version should pass all tests
@pytest.mark.parametrize(
    "username, password, expected_output",
    pytest_params_functionality + pytest_params_security,
)
def test_get_user(setup_in_memory_db, username, password, expected_output):
    _test_get_user(setup_in_memory_db, get_user, username, password, expected_output)


# The unsafe version should pass all functionality tests
@pytest.mark.parametrize(
    "username, password, expected_output", pytest_params_functionality
)
def test_get_user_unsafe_functionality(
    setup_in_memory_db, username, password, expected_output
):
    _test_get_user(
        setup_in_memory_db, get_user_unsafe, username, password, expected_output
    )


# The unsafe version should fail all security tests
@pytest.mark.parametrize("username, password, expected_output", pytest_params_security)
def test_get_user_unsafe_security(
    setup_in_memory_db, username, password, expected_output
):
    with pytest.raises(AssertionError):
        _test_get_user(
            setup_in_memory_db, get_user_unsafe, username, password, expected_output
        )
