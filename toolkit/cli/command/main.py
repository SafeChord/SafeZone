#!/usr/bin/env python3
import sys
import uuid
import logging

import typer  # type: ignore
import rich  # type: ignore

from utils.logging.baselogger import setup_logger

from config.settings import TOOL_NAME, TOOL_VERSION
from bin.client import DataflowClient, DBClient, HealthClient, TimeClient
from bin.context import global_context
from bin.command import command_handler

app = typer.Typer(
    help="SafeZone CLI tool to control Dataflow, DB, and System operations.",
    pretty_exceptions_enable=False,
)


# ---- Callback Functions ----
@app.callback()
def main(
    debug: bool = typer.Option(False, "--debug", "-d", help="Enable debug logging."),
    log_level: str = typer.Option(
        "WARNING", "--log-level", "-l", help="Set the logging level."
    ),
    log_output: str = typer.Option(
        "stderr", "--log-output", help="Set the logging output."
    ),
    output_format: str = typer.Option(
        "rich", "--output", "-o", help="Set the output format."
    ),
):
    # set context
    global_context.set("trace_id", str(uuid.uuid4()))
    global_context.set("output_format", output_format)

    # setup logger
    if debug:
        log_level = "DEBUG"

    setup_logger(
        log_level=log_level,
        log_output=log_output,
        service_name=TOOL_NAME,
        service_version=TOOL_VERSION,
    )


# ---- General Commands ----
@app.command("version")
def version():
    """Show the version information."""
    pass


@app.command("config")
def config():
    """Show the configuration information."""
    pass


# ---- Database Commands ----
db_app = typer.Typer(help="Database control commands (init, clear, reset).")


@db_app.command()
@command_handler("db.init")
def init(
    force: bool = typer.Option(
        False, "--force", help="Force re-initialize the database."
    ),
):
    """Initialize the covid data in the database."""
    return DBClient().init(force=force)


@db_app.command()
@command_handler("db.clear")
def clear():
    """Clear the covid data in the database."""
    if not typer.confirm("Are you sure you want to clear the database?"):
        rich.print("Aborted.")
        raise typer.Abort()

    return DBClient().clear()


@db_app.command()
@command_handler("db.reset")
def reset():
    """Reset the database to its initial state."""
    if not typer.confirm("Are you sure you want to reset the database?"):
        rich.print("Aborted.")
        raise typer.Abort()
    
    return DBClient().reset()


app.add_typer(db_app, name="db")

# -- Health Commands --
health_app = typer.Typer(help="Health check commands.")


@health_app.command("all")
@command_handler("health.all")
def all():
    """Check the health of all components."""
    return HealthClient().check(target="all")


@health_app.command("cli-relay")
@command_handler("health.cli-relay")
def cli_relay():
    """Check the health of the CLI relay component."""
    return HealthClient().check(target="cli-relay")


@health_app.command("db")
@command_handler("health.db")
def db():
    """Check the health of the database component."""
    return HealthClient().check(target="db")


@health_app.command("redis-state")
@command_handler("health.redis-state")
def redis_state():
    """Check the health of the Redis state component."""
    return HealthClient().check(target="redis-state")


@health_app.command("redis-cache")
@command_handler("health.redis-cache")
def redis_cache():
    """Check the health of the Redis cache component."""
    return HealthClient().check(target="redis-cache")


@health_app.command("simulator")
@command_handler("health.simulator")
def simulator():
    """Check the health of the simulator component."""
    return HealthClient().check(target="simulator")


@health_app.command("ingestor")
@command_handler("health.ingestor")
def ingestor():
    """Check the health of the ingestor component."""
    return HealthClient().check(target="ingestor")


@health_app.command("analytics-api")
@command_handler("health.analytics-api")
def analytics_api():
    """Check the health of the analytics API component."""
    return HealthClient().check(target="analytics-api")


@health_app.command("dashboard")
@command_handler("health.dashboard")
def dashboard():
    """Check the health of the dashboard component."""
    return HealthClient().check(target="dashboard")


app.add_typer(health_app, name="health")

# ---- Dataflow Commands ----
dataflow_app = typer.Typer(help="Dataflow control commands (simulate, verify).")


@dataflow_app.command()
@command_handler("dataflow.simulate")
def simulate(
    date: str = typer.Argument(..., help="Start date for simulation (YYYY-MM-DD)"),
    enddate: str = typer.Option(
        None, "--enddate", help="End date for interval simulation (YYYY-MM-DD)"
    ),
    dry_run: bool = typer.Option(False, "--dry-run", help="Dry run mode"),
):
    """Simulate covid data for a specific date or interval."""
    return DataflowClient().simulate(date=date, end_date=enddate, dry_run=dry_run)


@dataflow_app.command()
@command_handler("dataflow.verify")
def verify(
    date: str = typer.Argument(..., help="Date to verify (YYYY-MM-DD)"),
    interval: int = typer.Option(
        1, "--interval", help="Interval in days for verification"
    ),
    city: str = typer.Option(None, "--city", help="City"),
    region: str = typer.Option(None, "--region", help="Region"),
    ratio: bool = typer.Option(False, "--ratio", help="Show as ratio (per 10,000)"),
):
    """Verify covid data in the database."""
    return DataflowClient().verify(
        date=date, interval=interval, city=city, region=region, ratio=ratio
    )


app.add_typer(dataflow_app, name="dataflow")

# ---- System Commands ----
system_app = typer.Typer(help="System control commands (time control, health check).")

# -- Time Commands --
time_app = typer.Typer(help="Time control commands.")


@time_app.command("now")
@command_handler("time.now")
def now():
    """Get the current system time."""
    return TimeClient().now()


@time_app.command("set")
@command_handler("time.set")
def set(
    reset: bool = typer.Option(
        False, "--reset", help="Reset the system time to real time."
    ),
    mock_date: str = typer.Option(None, "--mockdate", help="Mock date (YYYY-MM-DD)"),
    acceleration: int = typer.Option(
        None, "--acceleration", help="Time acceleration rate"
    ),
):
    """Set the system mock_date or acceleration or reset to real time."""
    mock = False if reset else True

    # Check if both mock_date and acceleration are None
    if mock and not mock_date and not acceleration:
        rich.print(
            "[bold red]Error:[/bold red] You must specify either --mockdate or --acceleration."
        )
        raise typer.Exit(1)

    return TimeClient().set(mock=mock, mock_date=mock_date, acceleration=acceleration)


@time_app.command("status")
@command_handler("time.status")
def status():
    """Get current time management status."""
    return TimeClient().get_status()


system_app.add_typer(time_app, name="time")
app.add_typer(system_app, name="system")


if __name__ == "__main__":
    app()
