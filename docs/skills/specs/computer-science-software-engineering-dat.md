# 79.Y — Computer Science, Software Engineering, Databases e Advanced Linux Skills

> Authority: canonical specification.
> Logical ID: 79 Y
> Source: Notion Living Book (3dd9bb7d023f81fda05ccde2975d66cb)
> Status: Skill Package de referência para 79 Y — Computer Science, Software Engineering, Dat.


<aside>
🧠

**Meta:** o Prumo deve possuir knowledge de engenharia suficientemente profundo para distinguir “gerar código que funciona” de “ensinar e aplicar fundamentos de computação”. ACM CS2023 e SWEBOK v4 servem como mapas de cobertura, não como currículos rígidos.

</aside>

# Computer Science Knowledge Map

Basear coverage no ACM CS2023: algorithms; architecture/organization; AI; data management; foundations of programming languages; graphics; HCI; math/statistics; networking; operating systems; parallel/distributed; security; software development fundamentals; software engineering; specialized platforms; systems fundamentals.

## Skills candidatas

`algorithms-analysis`, `data-structures`, `discrete-math`, `computer-architecture`, `operating-systems-theory`, `networks-protocols`, `distributed-systems`, `concurrency-parallelism`, `information-theory`, `numerical-computing`, `graphics-foundations`, `computability-complexity`, `cryptography-foundations`.

Cada uma precisa de theory + implementation + measurement + failure modes + teaching profile.

# Software Engineering

Usar SWEBOK v4 como coverage map. Skills: requirements, architecture, design, construction, testing, operations, security, quality, maintenance/evolution, configuration management, engineering management, process, modeling/methods, economics, professional practice e foundations.

## Hardening especial

`architecture-quality`, `requirements-discovery`, `testing-quality`, `debugging-methodology`, `maintenance-engineering`, `release-engineering`, `observability`, `performance-analysis`, `reliability`, `incident-response`, `technical-debt`, `dependency-management`, `API/evolution`.

# Database Engineering family

Separar modelagem/desenvolvimento de DBA/operations.

## `database-design`

Relational model, normalization/denormalization, keys/constraints, transactions, isolation, schema evolution, relational algebra, query semantics e data modeling.

## `sql-query-engineering`

Dialect-aware SQL, joins/subqueries/window/CTE, indexes, planner/cardinality, EXPLAIN/ANALYZE, locking/concurrency, injection-safe parameterization e query regression.

## `database-administration`

Backup/restore **com restore tests**, replication/HA, WAL/binlog, PITR, vacuum/autovacuum/analyze, capacity, connection pools, roles/permissions, encryption, observability, upgrades, corruption/recovery e runbooks.

## Packs

`postgresql-engineering` P0/P1, `sqlite-engineering` P1, `mysql-mariadb-engineering` P1, mais KV/document/vector somente quando usados. PostgreSQL knowledge deve compreender MVCC, VACUUM, planner stats, WAL/LSN, physical/logical replication e lock analysis.

# Advanced Linux family

Linux não pode ser apenas shell/package manager.

## Core

process/thread model, syscalls, file descriptors, signals, virtual memory, ELF/dynamic loader, filesystems/VFS, permissions/capabilities, procfs/sysfs, networking stack, sockets, epoll, futex, IPC, scheduling, NUMA e resource limits.

## Isolation/security

Namespaces, cgroup v2, seccomp-BPF, Landlock, capabilities, no-new-privileges, LSM, sandboxing, container primitives e privilege boundaries.

## Performance/observability

`perf`, tracepoints/kprobes/uprobes, eBPF, bpftrace, ftrace, PSI, `/proc`, flame graphs, scheduler/I/O/network profiling e reproducible benchmark environments.

## Modern async I/O

io_uring, epoll/eventfd/timerfd/signalfd; ensinar trade-offs, lifetime/cancellation/resource ownership e kernel-version compatibility.

## systemd/operations

Unit lifecycle/dependencies, service sandboxing/hardening, resource control/cgroups, socket/timer/path activation, journald, credentials, restart semantics e daemon design.

## Kernel-adjacent

Kernel build/config, modules, KUnit/kselftest, QEMU, fault injection, driver basics, netlink/ioctl/uAPI design e ABI stability.

# Teaching integration

Toda skill teórica importante deve poder expor `teach_from_first_principles`, `worked_example`, `exercise`, `hint_ladder`, `visual_model`, `misconception_checks` e `transfer_problem`.

# Fontes

- ACM CS2023: [https://csed.acm.org/knowledge-areas/](https://csed.acm.org/knowledge-areas/)
- SWEBOK v4: [https://www.computer.org/education/bodies-of-knowledge/software-engineering](https://www.computer.org/education/bodies-of-knowledge/software-engineering)
- Linux kernel docs: [https://docs.kernel.org/](https://docs.kernel.org/)
- man-pages: [https://man7.org/linux/man-pages/](https://man7.org/linux/man-pages/)
- PostgreSQL: [https://www.postgresql.org/docs/](https://www.postgresql.org/docs/)
- bpftrace: [https://bpftrace.org/docs/](https://bpftrace.org/docs/)

# Exit Gate

Coverage é medido por domínio/competência/evals, não por quantidade de skills. O Prumo deve saber tanto implementar quanto explicar fundamentos e operações reais.