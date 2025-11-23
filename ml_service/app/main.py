from __future__ import annotations

from datetime import datetime
from typing import List

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field, field_validator

from .scoring import TodoFeatures, priority_score

app = FastAPI(
    title="Smart Todo Priority Service",
    summary="Assigns a normalized priority_score to todo items.",
    version="0.1.0",
)


class TodoPayload(BaseModel):
    title: str = Field(..., min_length=1, max_length=200)
    completed: bool = False
    created_at: datetime | None = None
    due_date: datetime | None = None
    tags: List[str] = Field(default_factory=list, max_items=20)

    @field_validator("tags", mode="before")
    @classmethod
    def normalize_tags(cls, value: List[str]) -> List[str]:
        if value is None:
            return []
        if isinstance(value, str):
            return [value]
        tags = []
        for item in value:
            if not item:
                continue
            tags.append(str(item).strip())
        return tags


class ScoreRequest(BaseModel):
    todos: List[TodoPayload] = Field(..., min_length=1, max_length=50)


class ScoreResult(TodoPayload):
    priority_score: float


class ScoreResponse(BaseModel):
    results: List[ScoreResult]


@app.get("/health", tags=["system"])
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/score", response_model=ScoreResponse, tags=["scoring"])
def score(request: ScoreRequest) -> ScoreResponse:
    if len(request.todos) == 0:
        raise HTTPException(status_code=400, detail="at least one todo is required")

    results: List[ScoreResult] = []
    for todo in request.todos:
        features = TodoFeatures(
            title=todo.title,
            completed=todo.completed,
            created_at=todo.created_at,
            due_date=todo.due_date,
            tags=todo.tags,
        )
        results.append(
            ScoreResult(
                title=todo.title,
                completed=todo.completed,
                created_at=todo.created_at,
                due_date=todo.due_date,
                tags=todo.tags,
                priority_score=priority_score(features),
            )
        )
    return ScoreResponse(results=results)

