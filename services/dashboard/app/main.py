import logging
import dash # type: ignore
import dash_bootstrap_components as dbc # type: ignore
from flask import jsonify # type: ignore

from utils.logging.baselogger import setup_logger

from layout.dashboard_layout import get_dashboard_layout
from callbacks.register import register_callbacks
from config.settings import SERVER_IP, SERVER_PORT, SERVICE_NAME, SERVICE_VERSION, LOG_LEVEL

def create_app():
    app = dash.Dash(
        __name__,
        routes_pathname_prefix="/dashboard/classic/",
        requests_pathname_prefix="/dashboard/classic/",
        external_stylesheets=[dbc.themes.FLATLY],
    )
    app.layout = get_dashboard_layout()
    register_callbacks(app)

    @app.server.route("/health")
    def health_check():
        return jsonify({"status": "healthy"}), 200

    return app


if __name__ == "__main__":
    setup_logger(
        service_name=SERVICE_NAME,
        service_version=SERVICE_VERSION,
        log_level=LOG_LEVEL,
    )
    logger = logging.getLogger(__name__)
    app = create_app()
    logger.info(f"Starting {SERVICE_NAME} version {SERVICE_VERSION}.",  extra={"event": "service_startup"})
    app.run_server(host=SERVER_IP, port=SERVER_PORT)
