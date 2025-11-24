FROM python:3.12-slim

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    MLflow_VERSION=2.15.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential curl \
    && pip install --upgrade pip setuptools \
    && pip install "mlflow==${MLflow_VERSION}" \
    && apt-get purge -y build-essential \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /srv

EXPOSE 5000

CMD ["mlflow", "server", "--host", "0.0.0.0", "--port", "5000", "--backend-store-uri", "sqlite:////mlflow/mlflow.db", "--default-artifact-root", "/mlflow/artifacts"]

