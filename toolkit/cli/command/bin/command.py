
import logging
from functools import wraps

import typer # type: ignore

from bin.presenter import Presenter
from bin.context import global_context

def command_handler(command_name: str):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            logger = logging.getLogger(__name__)
            output_format = global_context.get("output_format")
            trace_id = global_context.get("trace_id")

            try:
                # call the client and get the response
                response_data = func(*args, **kwargs)

                # separate headers from response body
                headers = response_data.pop("headers", {})

                task = {
                    "task": {
                        "name": command_name,
                        "trace_id": trace_id,
                    },
                    "response": response_data,
                }

                # include headers only when verbose is enabled
                if global_context.get("verbose"):
                    task["headers"] = headers

                Presenter(output_format).render(task)

            except Exception as e:
                logger.error(f"{command_name} failed: {e}", exc_info=True)
                raise typer.Exit(1)
        return wrapper
    return decorator