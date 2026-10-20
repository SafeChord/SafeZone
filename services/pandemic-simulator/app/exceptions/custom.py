from pydantic import ValidationError  # type: ignore


class ServiceValidationError(Exception):
    def __init__(self, exc: ValidationError):
        self.errors = exc.errors()
        super().__init__("Validation error in the service.")


class EmptyDataError(Exception):
    def __init__(self, message="No data available for the given date(s)."):
        super().__init__(message)