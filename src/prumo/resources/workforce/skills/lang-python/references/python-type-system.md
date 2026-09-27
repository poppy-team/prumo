# Modern Python 3.12+ Type System & Concurrency Architecture

## 1. Type Annotation Modernization
Python's typing ecosystem has transitioned from legacy `typing` module wrappers to first-class language syntax:

| Legacy (Python 3.8-) | Modern Standard (Python 3.10+) | Modern Type Parameter (Python 3.12+) |
|---|---|---|
| `from typing import List, Dict` | `list[str]`, `dict[str, int]` | Standard built-in collections |
| `from typing import Union, Optional` | `int | None`, `str | bytes` | Native union operator `|` |
| `T = TypeVar('T')` | `def identity[T](val: T) -> T:` | PEP 695 type parameter syntax |
| `type alias = int` | `type UserID = int` | PEP 695 `type` statement |

---

## 2. Structural Subtyping with Protocols
Rather than rigid nominal inheritance, use `typing.Protocol` to define verifiable compile-time interfaces:
```python
from typing import Protocol, runtime_checkable


@runtime_checkable
class Renderable(Protocol):
    def render(self, buffer: bytearray) -> int: ...


def draw_component(obj: Renderable, buf: bytearray) -> None:
    bytes_written = obj.render(buf)
```
Mypy verifies that any class providing `render(self, buffer: bytearray) -> int` conforms to `Renderable` without explicit subclassing.

---

## 3. High-Performance Immutability & Memory Footprint
Use `@dataclass(frozen=True, slots=True)` for domain models:
- **`frozen=True`**: Prevents runtime mutation (raises `FrozenInstanceError` on assignment), generates deterministic `__hash__` and `__eq__`.
- **`slots=True`**: Eradicates per-instance `__dict__` overhead, reducing object memory consumption by up to 60% and speeding up attribute access.

```python
from dataclasses import dataclass
from typing import Final

DEFAULT_TIMEOUT: Final = 30.0


@dataclass(frozen=True, slots=True)
class ExecutionConfig:
    endpoint: str
    max_retries: int = 3
    timeout_seconds: float = DEFAULT_TIMEOUT
```

---

## 4. Structured Concurrency with `asyncio.TaskGroup`
Python 3.11 introduced `asyncio.TaskGroup`, replacing error-prone manual task tracking:
```python
import asyncio


async def worker(task_id: int) -> str:
    await asyncio.sleep(0.01)
    return f"Result {task_id}"


async def run_parallel_jobs(job_count: int) -> list[str]:
    results: list[str] = []
    async with asyncio.TaskGroup() as tg:
        tasks = [tg.create_task(worker(i)) for i in range(job_count)]
    # All tasks are guaranteed complete or cancelled upon exiting context
    return [t.result() for t in tasks]
```
If any child task fails, the remaining sibling tasks are cancelled immediately, preventing task leakage.
