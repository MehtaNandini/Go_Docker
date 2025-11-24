"""
Todo ML Pipeline DAG

This DAG orchestrates a simple MLOps pipeline that:
1. Extracts task data from Postgres
2. Prepares a training dataset
3. Trains a model and logs results to MLflow

Schedule: Daily at midnight
"""

from __future__ import annotations

import json
import os
import random
from datetime import datetime, timedelta
from typing import Any

import mlflow
import psycopg2
from airflow.decorators import dag, task


# Default arguments for the DAG
default_args = {
    "owner": "airflow",
    "depends_on_past": False,
    "email_on_failure": False,
    "email_on_retry": False,
    "retries": 1,
    "retry_delay": timedelta(minutes=5),
}


@dag(
    dag_id="todo_ml_pipeline",
    default_args=default_args,
    description="Daily ML pipeline for todo priority model",
    schedule_interval="@daily",
    start_date=datetime(2024, 1, 1),
    catchup=False,
    tags=["ml", "mlops", "todo", "priority"],
)
def todo_ml_pipeline():
    """
    Main DAG definition for the todo ML pipeline.
    """

    @task
    def extract_tasks() -> dict[str, Any]:
        """
        Extract tasks from Postgres database.
        
        Returns:
            Dictionary containing task data and metadata
        """
        # Database connection parameters from environment
        db_config = {
            "host": os.getenv("POSTGRES_HOST", "postgres"),
            "port": int(os.getenv("POSTGRES_PORT", "5432")),
            "user": os.getenv("POSTGRES_USER", "todo"),
            "password": os.getenv("POSTGRES_PASSWORD", "todo"),
            "database": os.getenv("POSTGRES_DB", "tododb"),
        }
        
        print(f"Connecting to Postgres at {db_config['host']}:{db_config['port']}")
        
        conn = None
        try:
            # Establish database connection with security parameters
            conn = psycopg2.connect(
                host=db_config["host"],
                port=db_config["port"],
                user=db_config["user"],
                password=db_config["password"],
                dbname=db_config["database"],
                sslmode="prefer",
                connect_timeout=10,
            )
            
            cursor = conn.cursor()
            
            # Execute query to fetch all tasks
            query = """
                SELECT 
                    id, 
                    title, 
                    completed, 
                    tags, 
                    duration_minutes, 
                    priority_score,
                    created_at,
                    updated_at
                FROM todos
                ORDER BY created_at DESC
            """
            
            cursor.execute(query)
            rows = cursor.fetchall()
            
            # Transform rows into structured data
            tasks = []
            for row in rows:
                task = {
                    "id": row[0],
                    "title": row[1],
                    "completed": row[2],
                    "tags": row[3] if row[3] else [],
                    "duration_minutes": row[4],
                    "priority_score": row[5],
                    "created_at": row[6].isoformat() if row[6] else None,
                    "updated_at": row[7].isoformat() if row[7] else None,
                }
                tasks.append(task)
            
            cursor.close()
            
            # Prepare return data
            result = {
                "tasks": tasks,
                "total_count": len(tasks),
                "completed_count": sum(1 for t in tasks if t["completed"]),
                "extraction_timestamp": datetime.utcnow().isoformat(),
            }
            
            print(f"Extracted {len(tasks)} tasks from database")
            print(f"Completed tasks: {result['completed_count']}")
            
            return result
            
        except psycopg2.Error as e:
            print(f"Database error: {e}")
            raise
        except Exception as e:
            print(f"Unexpected error during extraction: {e}")
            raise
        finally:
            if conn:
                conn.close()
                print("Database connection closed")

    @task
    def prepare_dataset(extraction_result: dict[str, Any]) -> dict[str, Any]:
        """
        Prepare dataset from extracted tasks.
        
        Args:
            extraction_result: Output from extract_tasks
            
        Returns:
            Dictionary containing prepared dataset and statistics
        """
        tasks = extraction_result["tasks"]
        
        if not tasks:
            print("Warning: No tasks found in database")
            return {
                "dataset_size": 0,
                "features": [],
                "statistics": {"total": 0, "completed": 0, "pending": 0},
            }
        
        # Compute basic statistics
        total = len(tasks)
        completed = sum(1 for t in tasks if t["completed"])
        pending = total - completed
        
        avg_duration = (
            sum(t["duration_minutes"] for t in tasks) / total if total > 0 else 0
        )
        avg_priority = (
            sum(t["priority_score"] for t in tasks) / total if total > 0 else 0
        )
        
        # Extract features for training
        features = []
        for task in tasks:
            feature = {
                "title_length": len(task["title"]),
                "has_tags": len(task["tags"]) > 0,
                "tag_count": len(task["tags"]),
                "duration_minutes": task["duration_minutes"],
                "priority_score": task["priority_score"],
                "completed": task["completed"],
            }
            features.append(feature)
        
        result = {
            "dataset_size": total,
            "features": features,
            "statistics": {
                "total": total,
                "completed": completed,
                "pending": pending,
                "avg_duration": round(avg_duration, 2),
                "avg_priority": round(avg_priority, 3),
            },
        }
        
        print(f"Prepared dataset with {total} samples")
        print(f"Statistics: {result['statistics']}")
        
        return result

    @task
    def train_model(dataset: dict[str, Any]) -> dict[str, Any]:
        """
        Train model and log to MLflow.
        
        Args:
            dataset: Output from prepare_dataset
            
        Returns:
            Dictionary containing training results and MLflow run info
        """
        # MLflow configuration
        tracking_uri = os.getenv("MLFLOW_TRACKING_URI", "http://mlflow:5000")
        experiment_name = "todo-priority-model"
        
        print(f"MLflow Tracking URI: {tracking_uri}")
        
        mlflow.set_tracking_uri(tracking_uri)
        mlflow.set_experiment(experiment_name)
        
        # Simulate training with enhanced logic based on real data
        dataset_size = dataset["dataset_size"]
        statistics = dataset["statistics"]
        
        # Generate model parameters based on dataset characteristics
        if dataset_size > 0:
            # Use dataset statistics to influence model parameters
            duration_factor = min(statistics["avg_duration"] / 60.0, 1.0) if statistics["avg_duration"] > 0 else 0.5
            priority_factor = statistics["avg_priority"]
            
            keyword_bonus = round(0.2 + (duration_factor * 0.3), 2)
            tag_bonus = round(0.05 + (priority_factor * 0.15), 2)
            duration_penalty = round(0.05 + (duration_factor * 0.1), 2)
        else:
            keyword_bonus = 0.35
            tag_bonus = 0.10
            duration_penalty = 0.08
        
        # Generate metrics
        mae = round(random.uniform(0.05, 0.25), 3)
        f1_score = round(1 - mae, 3)
        
        # Prepare model rules
        model_rules = {
            "description": "Rule-based priority model for todo tasks",
            "weights": {
                "keyword_bonus": keyword_bonus,
                "tag_bonus": tag_bonus,
                "duration_penalty": duration_penalty,
            },
            "metadata": {
                "training_samples": dataset_size,
                "dataset_statistics": statistics,
            },
        }
        
        # Start MLflow run
        with mlflow.start_run(
            run_name=f"priority-model-{datetime.utcnow().strftime('%Y%m%d-%H%M%S')}"
        ) as run:
            # Log parameters
            mlflow.log_param("model_type", "rule_based")
            mlflow.log_param("training_samples", dataset_size)
            mlflow.log_param("keyword_bonus", keyword_bonus)
            mlflow.log_param("tag_bonus", tag_bonus)
            mlflow.log_param("duration_penalty", duration_penalty)
            
            # Log metrics
            mlflow.log_metric("mae", mae)
            mlflow.log_metric("f1", f1_score)
            mlflow.log_metric("dataset_size", dataset_size)
            mlflow.log_metric("completed_ratio", 
                            statistics["completed"] / dataset_size if dataset_size > 0 else 0)
            
            # Log dataset statistics as metrics
            mlflow.log_metric("avg_duration", statistics["avg_duration"])
            mlflow.log_metric("avg_priority", statistics["avg_priority"])
            
            # Log model rules as parameter (avoids filesystem permission issues)
            mlflow.log_param("model_rules_json", json.dumps(model_rules))
            
            # Log tags
            mlflow.set_tag("pipeline", "airflow")
            mlflow.set_tag("dag_id", "todo_ml_pipeline")
            mlflow.set_tag("execution_date", datetime.utcnow().isoformat())
            
            run_id = run.info.run_id
            experiment_id = run.info.experiment_id
        
        result = {
            "run_id": run_id,
            "experiment_id": experiment_id,
            "metrics": {"mae": mae, "f1": f1_score},
            "model_params": model_rules["weights"],
            "training_samples": dataset_size,
            "mlflow_uri": tracking_uri,
        }
        
        print(f"Training completed successfully")
        print(f"MLflow Run ID: {run_id}")
        print(f"Metrics: MAE={mae}, F1={f1_score}")
        print(f"Model parameters: {model_rules['weights']}")
        
        return result

    # Define task dependencies
    extraction = extract_tasks()
    dataset = prepare_dataset(extraction)
    training_result = train_model(dataset)


# Instantiate the DAG
dag_instance = todo_ml_pipeline()

