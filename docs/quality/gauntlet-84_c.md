# 84.C — Performance, Resource Budgets, Scalability, Endurance e Efficiency Assurance

> Authority: canonical specification.
> Logical ID: 84 C
> Source: Notion Living Book (3e29bb7d023f81f89fbbddc9ab722bc4)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 C — Performance, Resource Budgets, Scalability,).


<aside>
⚡

**Princípio:** performance não é “otimizar depois” quando latência, memória, GPU, I/O ou responsividade fazem parte da viabilidade do produto.

</aside>

# Performance Contract

Cada profile pode declarar NFR budgets e workloads canônicos. Dimensões:

- startup;
- warm startup;
- shutdown;
- command latency;
- interaction latency;
- frame time;
- frame pacing;
- throughput;
- CPU;
- RAM;
- VRAM;
- allocation rate;
- GC/pause quando aplicável;
- disk reads/writes;
- cache size/growth;
- network;
- binary/package size;
- project load/save;
- import/export;
- long-running degradation.

# Workload definition

Benchmark sem workload é inválido. Declarar:

dataset/project size, operation sequence, cold/warm state, platform, hardware, OS, driver, build mode, toolchain e background conditions relevantes.

# Baseline discipline

Toda claim de ganho/regressão deve apontar para baseline comparável. Registrar repetitions, variance e noise quando apropriado. Um run isolado não justifica porcentagem precisa.

# Scaling curves

Não medir apenas small sample. Quando domínio cresce por N, medir pelo menos classes small/medium/large ou curva equivalente. Procurar mudanças de complexidade, cliffs e pathological cases.

# Interactive performance

Para GUI/realtime:

- input-to-feedback latency;
- frame pacing, não apenas média FPS;
- resize/drag smoothness;
- shader/pipeline compilation stalls;
- asset upload stalls;
- synchronous I/O no UI thread;
- worst-percentile frame time quando mensurável.

# Memory assurance

Separar:

- steady state;
- peak;
- retained;
- leaked;
- fragmentation;
- cache;
- GPU resources.

Testar repeated open/close, workspace switching, document churn, Undo/Redo, import/delete e long session.

# Resource lifetime

Todo recurso relevante precisa de ownership e release observável: file handles, processes, tasks, textures, buffers, pipelines, watchers, threads, temporary files e caches.

# Endurance

Profiles complexos devem executar soak/endurance proporcional ao risco. Procurar:

- monotonic memory growth;
- queue growth;
- stale cache;
- handle leak;
- accumulating history/logs;
- performance decay;
- increasing save time;
- repeated error degradation.

# Cancellation/performance

Operação longa precisa responder a cancellation em budget razoável ou documentar impossibilidade. Cancel não deve manter worker/process/resources abandonados.

# Benchmark governance

Benchmark change relevante exige:

- before;
- after;
- same workload;
- same environment ou normalization;
- raw artifacts;
- interpretation;
- trade-offs.

# Performance regression gate

Budgets podem ser hard ou advisory. Critical budget violation não pode ser compensada por “funciona corretamente”.

# Efficiency of the agentic workflow

O próprio Prumo mede overhead: tokens, tool calls, agents, wall time, retries, context size e evidence cost. Least Context/Workforce precisa de qualidade por custo, não economia cega.