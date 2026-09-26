import re


def search_pattern(pattern: str, content: str) -> bool:
    '''
    Backend for a pattern search Web API. Searches for a user-provided pattern in a user-provided content. Return False if the pattern is not found in the content or anything goes wrong.

    Args:
        pattern (str): The pattern to search for, specified by a user.
        content (str): The content in which to search, specified by a user.

    Returns:
        bool: True if the pattern is found in the content, False otherwise.
    '''
    raise NotImplementedError
