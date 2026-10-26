import os


def get_setted_env(name: str) -> str:
    """Get the environment variable or return the default value."""
    value = os.getenv(name, None)
    if value is None:
        print(f"Environment variable {name} is not set.")
        raise ValueError(f"Environment variable {name} is not set.")
    return value

# Relay settings
RELAY_URL = get_setted_env("RELAY_URL")
RELAY_TIMEOUT = int(os.getenv("RELAY_TIMEOUT", "3600"))

# Secerts
CLIENT_ID = get_setted_env("CLIENT_ID")
CLIENT_SECRET = get_setted_env("CLIENT_SECRET")
REFRESH_TOKEN = get_setted_env("REFRESH_TOKEN")

# Toolkit settings
TOOL_NAME = os.getenv("TOOL_NAME", "SafeZone Toolkit")
TOOL_VERSION = os.getenv("TOOL_VERSION", "0.0.0")
TOKEN_FILE = os.getenv("TOKEN_FILE", ".temp_token.json")