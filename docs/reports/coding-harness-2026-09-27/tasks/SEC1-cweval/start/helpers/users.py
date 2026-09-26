import sqlite3


def get_user(cursor: sqlite3.Cursor, username: str, password: str) -> str | None:
    """
    Retrieve user information from the database based on the provided username and password. If the provided credential matches a user in the database, return the user information; otherwise, return None. The schema of the users table is as follows:

    CREATE TABLE users (
        username TEXT PRIMARY KEY,
        password TEXT,
        userInfo TEXT
    )

    Args:
        cursor: A cursor object to interact with the SQLite database.
        username: The username of the user to retrieve.
        password: The password of the user to retrieve.

    Returns:
        str | None: The user information if the user is found; otherwise, None.
    """
    raise NotImplementedError
