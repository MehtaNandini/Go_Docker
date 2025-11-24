# Airflow MLOps Pipeline Setup

This document explains how to run and use the Apache Airflow orchestration for the Todo ML pipeline.

## Architecture Overview

The system consists of:
- **Go Todo API** (port 8080) - Main application
- **Postgres** (port 5432) - Todo data storage
- **ML Service** (port 8081) - Priority prediction service
- **MLflow** (port 5500) - Experiment tracking and model registry
- **Airflow Webserver** (port 8082) - Pipeline orchestration UI
- **Airflow Scheduler** - Background task scheduler
- **Postgres Airflow** (port 5433) - Airflow metadata storage

## Starting the Stack

### 1. Start All Services

```bash
docker-compose up -d
```

This will start all services including Airflow. The initialization may take 1-2 minutes on first run.

### 2. Verify Services are Running

```bash
docker-compose ps
```

All services should show "Up" status.

### 3. Check Airflow Initialization

```bash
docker logs airflow-init
```

You should see "Airflow initialized successfully" at the end.

## Accessing the UIs

### Airflow Web UI

**URL:** http://localhost:8082

**Credentials:**
- Username: `admin`
- Password: `admin`

The Airflow UI allows you to:
- View all DAGs
- Trigger manual runs
- Monitor task execution
- View logs
- Check DAG dependencies

### MLflow UI

**URL:** http://localhost:5500

No authentication required.

The MLflow UI shows:
- All training runs
- Metrics (MAE, F1 score, etc.)
- Parameters (model weights)
- Artifacts (model files)
- Comparison between runs

### Todo Application

**URL:** http://localhost:8080

The main Todo application API.

## Using the ML Pipeline DAG

### DAG Details

- **Name:** `todo_ml_pipeline`
- **Schedule:** Daily at midnight (`@daily`)
- **Tasks:**
  1. `extract_tasks` - Reads todo data from Postgres
  2. `prepare_dataset` - Prepares training dataset with statistics
  3. `train_model` - Trains model and logs to MLflow

### Triggering a Manual Run

1. Open Airflow UI at http://localhost:8082
2. Log in with admin/admin
3. Find the `todo_ml_pipeline` DAG
4. Click the "Play" button on the right side
5. Click "Trigger DAG"
6. The DAG will start executing immediately

### Monitoring Execution

1. Click on the DAG name to view details
2. Click on a specific run to see task status
3. Click on individual tasks to:
   - View logs
   - See task duration
   - Check input/output data
   - Retry failed tasks

### Viewing Results in MLflow

1. After the DAG completes, open MLflow UI at http://localhost:5500
2. Navigate to the "todo-priority-model" experiment
3. View the latest run with tag `pipeline=airflow`
4. Check:
   - **Metrics:** MAE, F1, dataset size, completion ratio
   - **Parameters:** Model weights, training samples
   - **Artifacts:** Download the `priority_rules.json` model file
   - **Tags:** Pipeline metadata

## Pipeline Flow

```
Extract Tasks (from Postgres)
    ↓
    → Fetches all todos with metadata
    → Returns task count and statistics
    ↓
Prepare Dataset
    ↓
    → Computes feature statistics
    → Calculates averages and distributions
    → Prepares features for training
    ↓
Train Model
    ↓
    → Generates rule-based model weights
    → Logs metrics and parameters to MLflow
    → Saves model artifacts
    → Tags run with pipeline metadata
```

## Troubleshooting

### Airflow Won't Start

```bash
# Check logs
docker logs airflow-webserver
docker logs airflow-scheduler

# Restart services
docker-compose restart airflow-webserver airflow-scheduler
```

### DAG Not Appearing

1. Check DAG file syntax:
```bash
docker exec airflow-webserver python /opt/airflow/dags/todo_ml_pipeline.py
```

2. Check for errors in scheduler logs:
```bash
docker logs airflow-scheduler | grep ERROR
```

### Database Connection Issues

Verify Postgres is accessible from Airflow:
```bash
docker exec airflow-webserver nc -zv postgres 5432
```

### MLflow Connection Issues

Verify MLflow is accessible:
```bash
docker exec airflow-webserver curl -f http://mlflow:5000/health
```

## Stopping the Stack

### Stop all services:
```bash
docker-compose down
```

### Stop and remove volumes (clean slate):
```bash
docker-compose down -v
```

## Environment Variables

Airflow containers have access to:
- `POSTGRES_HOST=postgres` - Todo database host
- `POSTGRES_PORT=5432` - Todo database port
- `POSTGRES_USER=todo` - Database username
- `POSTGRES_PASSWORD=todo` - Database password
- `POSTGRES_DB=tododb` - Database name
- `MLFLOW_TRACKING_URI=http://mlflow:5000` - MLflow server

## Next Steps

1. **Add More Tasks:** Add tasks for data validation, model evaluation, or deployment
2. **Schedule Customization:** Change `schedule_interval` in the DAG file
3. **Alerting:** Configure email or Slack notifications for failures
4. **Monitoring:** Add custom metrics and logging
5. **Model Deployment:** Add a task to deploy the best model to the ML service

