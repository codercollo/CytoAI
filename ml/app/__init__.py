from __future__ import annotations

import logging

from flask import Flask

from ml.app.config import Config
from ml.app.models.battery_health_model import BatteryHealthModel
from ml.app.models.repayment_risk_model import RepaymentRiskModel
from ml.app.routes.anomaly_routes import anomaly_bp
from ml.app.routes.health_routes import health_bp
from ml.app.routes.score_routes import score_bp
from ml.app.services.scoring_service import ScoringService


def create_app(config: type[Config] = Config) -> Flask:
    logging.basicConfig(level=logging.INFO)
    app = Flask(__name__)
    app.config.from_object(config)

    battery_model = BatteryHealthModel(config.BATTERY_MODEL_PATH)
    repayment_model = RepaymentRiskModel(config.REPAYMENT_MODEL_PATH)
    repayment_swap_model = RepaymentRiskModel(config.REPAYMENT_SWAP_MODEL_PATH)
    battery_model.load()
    repayment_model.load()
    repayment_swap_model.load()

    scoring_service = ScoringService(
        battery_model=battery_model,
        repayment_model=repayment_model,
        repayment_swap_model=repayment_swap_model,
        bhi_weight=config.BHI_WEIGHT,
        rri_weight=config.RRI_WEIGHT,
    )
    app.config["SCORING_SERVICE"] = scoring_service

    app.register_blueprint(health_bp)
    app.register_blueprint(score_bp)
    app.register_blueprint(anomaly_bp)

    return app