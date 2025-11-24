from __future__ import annotations

import json
import os
import random
from pathlib import Path
from typing import Any, Dict

import mlflow

EXPERIMENT_NAME = "todo-priority-model"
ARTIFACT_TMP_DIR = Path("ml_training") / "tmp_artifacts"


def simulate_training() -> Dict[str, Any]:
    """Return fake training outputs for a rule-based priority model."""
    mae = round(random.uniform(0.05, 0.25), 3)
    weights = {
        "keyword_bonus": round(random.uniform(0.2, 0.5), 2),
        "tag_bonus": round(random.uniform(0.05, 0.2), 2),
        "duration_penalty": round(random.uniform(0.05, 0.15), 2),
    }
    return {
        "metrics": {"mae": mae, "f1": round(1 - mae, 3)},
        "rules": {
            "description": "Simple heuristic priority model for todo tasks.",
            "weights": weights,
        },
    }


def main() -> None:
    # Use file-based tracking so both client and server access the same mlruns directory
    tracking_uri = os.getenv("MLFLOW_TRACKING_URI", "./mlruns")
    mlflow.set_tracking_uri(tracking_uri)
    mlflow.set_experiment(os.getenv("MLFLOW_EXPERIMENT_NAME", EXPERIMENT_NAME))

    training_output = simulate_training()

    with mlflow.start_run(run_name=os.getenv("MLFLOW_RUN_NAME", "rule-based-priority")):
        mlflow.log_param("model_type", "rule_based")
        mlflow.log_param("training_samples", 500)
        mlflow.log_metric("mae", training_output["metrics"]["mae"])
        mlflow.log_metric("f1", training_output["metrics"]["f1"])

        ARTIFACT_TMP_DIR.mkdir(parents=True, exist_ok=True)
        artifact_file = ARTIFACT_TMP_DIR / "priority_rules.json"
        artifact_file.write_text(json.dumps(training_output["rules"], indent=2, sort_keys=True))
        mlflow.log_artifact(str(artifact_file), artifact_path="model")

    print(f"Logged run to {tracking_uri}. Check the MLflow UI to inspect details.")


if __name__ == "__main__":
    main()

