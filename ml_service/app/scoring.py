from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Iterable

KEYWORD_WEIGHTS = {
    "urgent": 0.35,
    "asap": 0.3,
    "important": 0.25,
    "today": 0.2,
    "tomorrow": 0.15,
    "email": 0.05,
    "call": 0.05,
}

TAG_WEIGHTS = {
    "work": 0.1,
    "home": 0.05,
    "bug": 0.2,
    "feature": 0.15,
}


@dataclass(frozen=True, slots=True)
class TodoFeatures:
    title: str
    completed: bool = False
    created_at: datetime | None = None
    due_date: datetime | None = None
    tags: Iterable[str] = field(default_factory=tuple)


def priority_score(features: TodoFeatures) -> float:
    """Return a normalized priority score in [0, 1]."""
    score = 0.35
    score += _keyword_bonus(features.title)
    score += _tag_bonus(features.tags)
    score += _age_bonus(_normalize_dt(features.created_at))
    score += _due_date_bonus(_normalize_dt(features.due_date))

    if features.completed:
        score -= 0.6

    return max(0.0, min(1.0, round(score, 3)))


def _keyword_bonus(title: str) -> float:
    title_lc = title.lower()
    bonus = 0.0
    for keyword, weight in KEYWORD_WEIGHTS.items():
        if keyword in title_lc:
            bonus += weight
    if len(title_lc.split()) >= 8:
        bonus += 0.05
    return min(bonus, 0.45)


def _tag_bonus(tags: Iterable[str]) -> float:
    if not tags:
        return 0.0
    bonus = 0.0
    for raw in tags:
        tag = raw.lower().strip()
        bonus += TAG_WEIGHTS.get(tag, 0.03)
    return min(bonus, 0.25)


def _age_bonus(created_at: datetime | None) -> float:
    if created_at is None:
        return 0.0
    age_hours = (datetime.now(timezone.utc) - created_at).total_seconds() / 3600
    if age_hours <= 24:
        return 0.05
    if age_hours <= 24 * 3:
        return 0.1
    if age_hours <= 24 * 7:
        return 0.15
    return 0.2


def _due_date_bonus(due_date: datetime | None) -> float:
    if due_date is None:
        return 0.0
    delta_hours = (due_date - datetime.now(timezone.utc)).total_seconds() / 3600
    if delta_hours <= 0:
        return 0.3
    if delta_hours <= 24:
        return 0.25
    if delta_hours <= 24 * 3:
        return 0.2
    if delta_hours <= 24 * 7:
        return 0.1
    return 0.05


def _normalize_dt(value: datetime | None) -> datetime | None:
    if value is None:
        return None
    if value.tzinfo is None:
        return value.replace(tzinfo=timezone.utc)
    return value.astimezone(timezone.utc)


__all__ = ["TodoFeatures", "priority_score"]

