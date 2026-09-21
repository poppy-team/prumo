# 79.Y — Programming Languages, Computer Science, Software Engineering, Databases e Linux Advanced Skills

> Authority: canonical specification.
> Logical ID: 79 Y
> Source: Notion Living Book (3de9bb7d023f81d38bf8ecf60419e56a)
> Status: Skill Package de referência para 79 Y — Programming Languages, Computer Science, So.


<aside>
🧠

**Objetivo:** fazer o Prumo raciocinar com fundamentos de computação e engenharia, não apenas com frameworks. O catálogo deve poder compor conhecimento acadêmico, systems knowledge e operational practice conforme o problema.

</aside>

# Programming Language Engineering Suite

A família de desenvolvimento de linguagens deve abranger **paradigmas e filosofias**, não somente lexer/parser/backend.

## Paradigms / semantic knowledge

- imperative/procedural;
- object-oriented e prototype-based;
- functional/pure/effectful;
- logic/relational;
- data-oriented;
- actor/message-passing;
- reactive/dataflow;
- array programming;
- concatenative/stack languages;
- systems programming;
- scripting/dynamic;
- declarative/DSL;
- concurrent/parallel/distributed models.

## Design philosophy skills

- `language-design`: goals, target users, novelty budget, orthogonality, learnability, consistency;
- `type-system-design`: static/dynamic, nominal/structural, inference, subtyping, generics, traits/typeclasses, algebraic types, dependent/refinement concepts quando relevante;
- `effect-error-design`: exceptions/results/effects/optionals/panics;
- `ownership-memory-design`: tracing GC, RC/ARC/ORC, ownership/borrowing, regions/arenas, linear/affine types, manual memory;
- `concurrency-language-design`: threads, async, actors, CSP, structured concurrency;
- `metaprogramming-design`: macros, templates, comptime, reflection, staging;
- `module-package-design`: namespaces/modules/packages/versioning;
- `ffi-abi-design`;
- `diagnostic-design`: source spans, recovery, actionable errors, suggestions;
- `language-evolution`: compatibility, deprecation, editions/features/versioning.

## Compiler/interpreter pipeline skills

`lexer`, Pratt/recursive-descent/LR/PEG parsing, AST/CST, name resolution, type checking/inference, desugaring, Core IR, SSA/CFG/dataflow, optimization, bytecode VM, register VM, JIT/AOT, GC/runtime, native backend, linker/object emission, debugger/LSP/formatter/package manager.

LLVM IR deve ser knowledge próprio; MLIR merece pack próprio para dialects, regions/blocks/ops, traits/interfaces, pattern rewriting, canonicalization, lowering e verification. MLIR explicitamente recomenda IR specs, verifiers, textual round-trip e FileCheck-style testing.

# Language Development Agents

Não criar agent por paradigma. Criar papéis distintos:

- `language-designer`: semantics/user model/evolution;
- `compiler-architect`: IR/pipeline/backend/runtime boundaries;
- `type-systems-reviewer`: soundness/ergonomics/diagnostics;
- `runtime-memory-engineer`: GC/RC/ownership/allocator/runtime;
- `language-tooling-engineer`: LSP/formatter/debugger/package/toolchain;
- `language-verification-reviewer`: differential/metamorphic/property/fuzz/conformance.

# Computer Science Foundation Skills

Usar **CS2023 ACM/IEEE-CS/AAAI** como mapa de cobertura. As 17 knowledge areas devem informar bundles, não virar obrigatoriamente 17 megaprompts:

- Algorithmic Foundations;
- Architecture and Organization;
- Artificial Intelligence;
- Data Management;
- Foundations of Programming Languages;
- Graphics and Interactive Techniques;
- Human-Computer Interaction;
- Mathematical and Statistical Foundations;
- Networking and Communication;
- Operating Systems;
- Parallel and Distributed Computing;
- Security;
- Society/Ethics/Profession;
- Software Development Fundamentals;
- Software Engineering;
- Specialized Platform Development;
- Systems Fundamentals.

Propor `cs-foundations` como router/knowledge graph, ativando micro-skills como algorithms/data-structures, complexity, automata/formal-languages, discrete-math, probability/statistics, architecture, OS, networking e distributed systems.

# Software Engineering Suite

Além de `clean-code`, adicionar/aprofundar:

- requirements engineering;
- architecture styles + tradeoff analysis;
- modularity/coupling/cohesion;
- API design;
- domain modeling;
- state machines;
- concurrency architecture;
- reliability/resilience;
- configuration management;
- maintenance/evolution;
- technical debt/refactoring;
- code review;
- testing strategy;
- performance engineering;
- release/deployment;
- observability;
- incident/postmortem;
- software economics/estimation sem falsa precisão;
- ethics/professional practice.

# Database Engineering + DBA

Separar `database-engineering` de `database-operations`.

## Database engineering

- relational model, keys/constraints/normalization/denormalization;
- transactions, ACID, isolation/anomalies, MVCC/locking;
- indexes/B-tree/hash/LSM concepts;
- query planning/optimization/cardinality/statistics;
- schema design/migrations;
- connection pooling/prepared statements;
- data lifecycle/retention/privacy;
- distributed DB/CAP/consistency quando aplicável;
- NoSQL/document/key-value/column/graph/time-series profiles.

## DBA / operations

- backup/restore proof;
- PITR/WAL/binlog;
- replication/failover;
- vacuum/analyze/maintenance;
- storage/capacity;
- monitoring/slow query analysis;
- permissions/roles/auditing;
- encryption/secrets;
- upgrades/migrations/rollback;
- HA/DR/RPO/RTO.

## Engine packs

- `database-postgresql`: PostgreSQL 18 current docs, MVCC, SSI, locks, autovacuum, EXPLAIN, WAL/replication;
- `database-sqlite`: pager/B-tree/VDBE/query planner/WAL/concurrency/embedding;
- MySQL/MariaDB only if portfolio usage justifies dedicated packs.

# Linux Advanced Suite

Não limitar Linux a shell/administração básica.

## User-space systems

process model, signals, files/fds, mmap, epoll, io_uring, procfs/sysfs, terminals/PTY, dynamic linking, capabilities, namespaces/cgroups/seccomp, IPC, sockets, scheduling/affinity, NUMA basics.

## Kernel/development

kernel build/config, modules, drivers, locking/concurrency, memory management, VFS, networking, tracing, BPF/eBPF, KUnit/kselftest, fault injection, crash/debugging, kernel security, patch workflow.

## Observability/performance

perf, ftrace, tracepoints, eBPF/BTF/libbpf, flamegraphs, pressure/IO/CPU/memory diagnostics, coredump, strace, systemtap/bpftrace when appropriate.

## Isolation/security

Linux capabilities instead of monolithic root assumptions; namespaces/cgroups/seccomp; LSM awareness; least privilege; container internals.

`io_uring` merece knowledge próprio em systems/network/storage projects: submission/completion rings compartilhados reduzem syscall/copy overhead, mas lifetime/cancellation/backpressure exigem cuidado.

# Referências

- CS2023: [https://csed.acm.org/](https://csed.acm.org/)
- LLVM: [https://llvm.org/docs/](https://llvm.org/docs/)
- MLIR: [https://mlir.llvm.org/docs/](https://mlir.llvm.org/docs/)
- Crafting Interpreters: [https://craftinginterpreters.com/](https://craftinginterpreters.com/)
- Linux Kernel docs: [https://docs.kernel.org/](https://docs.kernel.org/)
- Linux man-pages: [https://man7.org/linux/man-pages/](https://man7.org/linux/man-pages/)
- PostgreSQL 18: [https://www.postgresql.org/docs/18/](https://www.postgresql.org/docs/18/)
- SQLite architecture: [https://sqlite.org/arch.html](https://sqlite.org/arch.html)

# Exit Gate

O resolver consegue explicar quais fundamentos acadêmicos/engenharia/systems foram ativados; database design e DBA têm owners diferentes; Linux avançado cobre userspace+kernel+observability+isolation; language projects ativam semantic/paradigm skills além do pipeline mecânico.