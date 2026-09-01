from __future__ import annotations

from flask import Blueprint, current_app, jsonify

health_bp = Blueprint("health", __name__)


@health_bp.get("/healthz")
def healthz():
    """Liveness — process is up. Always 200 if the app is responding at all."""
    return jsonify({"status": "ok"}), 200


@health_bp.get("/readyz")
def readyz():
    """Readiness — both model artifacts are loaded and able to serve predictions."""
    scoring = current_app.config["SCORING_SERVICE"]
    battery_ready = scoring.battery_model.is_loaded
    repayment_ready = scoring.repayment_model.is_loaded
    ready = battery_ready and repayment_ready

    body = {
        "status": "ready" if ready else "not_ready",
        "battery_model_loaded": battery_ready,
        "repayment_model_loaded": repayment_ready,
    }
    return jsonify(body), (200 if ready else 503)