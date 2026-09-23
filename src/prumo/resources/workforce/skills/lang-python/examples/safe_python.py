#!/usr/bin/env python3
"""Modern Python 3.12+ Idiomatic Typing & Structured Concurrency Reference."""

import asyncio
from dataclasses import dataclass
from typing import Final, Protocol


# 1. Custom Domain Exception Hierarchy
class DomainError(Exception):
    """Base exception for domain-level errors."""


class ValidationError(DomainError):
    """Raised when input validation fails."""


# 2. Structural Subtyping with Protocol
class TaskProcessor(Protocol):
    def process(self, payload: str) -> str: ...


# 3. High-Performance Immutable Domain Entity
@dataclass(frozen=True, slots=True)
class WorkItem:
    item_id: str
    content: str
    priority: int = 1

    def __post_init__(self) -> None:
        if not self.item_id.strip():
            raise ValidationError("item_id cannot be empty")


# 4. Standard Processor Implementing Protocol
class UpperCaseProcessor:
    def process(self, payload: str) -> str:
        return payload.upper()


# 5. Generic Pipeline Function
def transform_payload[T: str](processor: TaskProcessor, data: T) -> str:
    return processor.process(data)


# 6. Structured Async Worker Loop using TaskGroup
async def execute_task(item: WorkItem, processor: TaskProcessor) -> str:
    await asyncio.sleep(0.01)
    transformed = transform_payload(processor, item.content)
    return f"Processed[{item.item_id}]: {transformed}"


async def run_pipeline(items: list[WorkItem]) -> list[str]:
    processor: Final = UpperCaseProcessor()
    results: list[str] = []

    async with asyncio.TaskGroup() as tg:
        tasks = [tg.create_task(execute_task(item, processor)) for item in items]

    for t in tasks:
        results.append(t.result())
    return results


def main() -> None:
    print("=== Modern Python 3.12+ Systems Demonstration ===")
    test_items = [
        WorkItem("item-1", "prumo framework", priority=2),
        WorkItem("item-2", "strict type soundness", priority=1),
        WorkItem("item-3", "structured concurrency", priority=3),
    ]

    output = asyncio.run(run_pipeline(test_items))
    for line in output:
        print(f"  {line}")


if __name__ == "__main__":
    main()
