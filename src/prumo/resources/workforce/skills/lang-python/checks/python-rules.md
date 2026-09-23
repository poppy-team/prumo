# Python Code Quality & Typing Invariants Checklist

## 1. Static Typing & Soundness
- [ ] Every function, method, and generator declares explicit parameter and return types.
- [ ] Zero unconstrained `typing.Any` without immediate runtime narrowing.
- [ ] No `# type: ignore` without an explanatory comment and ticket reference.
- [ ] Passes `mypy --strict` with `disallow_untyped_defs = true` and `no_implicit_optional = true`.
- [ ] Generic types use modern syntax (PEP 585 built-ins: `list[str]`, `dict[str, int]`, PEP 604 unions: `int | None`).

## 2. Exception & Error Handling
- [ ] Zero bare `except:` clauses.
- [ ] Zero swallowed exceptions (`except Exception: pass` without structured logging or re-raise).
- [ ] All custom domain exceptions inherit from a common base `DomainError(Exception)`.
- [ ] Exception chaining preserves original root cause: `raise DomainError(...) from err`.

## 3. Concurrency & Asynchronous Hygiene
- [ ] Concurrent asynchronous tasks are bound within an `asyncio.TaskGroup` context.
- [ ] No fire-and-forget `asyncio.create_task()` without strong reference ownership and awaitable lifecycle.
- [ ] `asyncio.CancelledError` is not swallowed; cleanup executes within `finally` or async context managers.

## 4. Security & Serialization Invariants
- [ ] Zero usage of `eval()` or `exec()`.
- [ ] Zero untrusted deserialization via `pickle`.
- [ ] Subprocess execution uses `shell=False`, arguments passed as `list[str]`, with explicit `timeout` defined.
- [ ] File operations validate paths against directory traversal using `Path.resolve().is_relative_to(base_dir)`.

## 5. Resource Management & Tooling
- [ ] All files, locks, database connections, and sessions are managed via `with` or `async with` blocks.
- [ ] Clean linting and formatting via `ruff check` and `ruff format`.
- [ ] Unit tests written with `pytest` and typed fixtures.
