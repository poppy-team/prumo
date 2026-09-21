# 79.X — Programming Language Engineering, Paradigms, Compiler Theory e Language Design Skills

> Authority: canonical specification.
> Logical ID: 79 X
> Source: Notion Living Book (3dd9bb7d023f81dca878cb14e2324a41)
> Status: Skill Package de referência para 79 X — Programming Language Engineering, Paradigms.


<aside>
🧪

O Prumo deve conseguir não apenas implementar compiladores, mas raciocinar sobre **semântica, paradigmas, trade-offs de linguagem, runtime, tooling e pedagogia da linguagem**. Essas capacidades devem ser ortogonais ao backend escolhido.

</aside>

# Taxonomia proposta

## Foundations

`formal-languages`, `grammars-parsing`, `automata`, `type-systems`, `operational-semantics`, `denotational-semantics`, `logic-proofs`, `program-analysis`, `abstract-interpretation`, `dataflow-analysis`.

## Frontend de linguagem

Lexing; recursive descent/Pratt/LR/PEG/GLR quando apropriados; CST vs AST; name resolution; scopes; symbol tables; desugaring; type checking/inference; overload resolution; diagnostics/recovery; source maps/spans.

## IR / middle-end

SSA/CFG, dominance, liveness, dataflow, constant propagation/folding, inlining, escape analysis, alias analysis, effect analysis, monomorphization/specialization, representation lowering e optimization validation.

## Backend/runtime

Bytecode VMs, stack/register VMs, interpreters, threaded dispatch, JIT/AOT, LLVM, Cranelift, native object emission, WASM backend, GC, reference counting, ARC/ORC, tracing/generational/incremental/concurrent GC, regions/arenas, ownership/borrowing, exceptions/unwinding, coroutines/async, FFI/ABI.

# Paradigm Skills

Criar knowledge/skills para comparar sem dogma:

- imperative/procedural;
- structured programming;
- object-oriented + prototype-based;
- functional/pure/impure;
- algebraic data types/pattern matching;
- logic/relational;
- data-oriented design;
- actor/CSP/message passing;
- reactive/dataflow/FRP;
- generic/metaprogramming/comptime;
- concatenative/stack-based;
- array/APL-style;
- ownership/affine/linear types;
- capability-oriented design;
- gradual/dynamic/static/dependent/refinement typing;
- effect systems and algebraic effects.

Cada skill deve explicar **quando o paradigma simplifica o domínio e quando aumenta complexidade**.

# Language Design Agent/Skills

Adicionar `language-architect` ou fortalecer `compiler-engineer` com um role distinto de design semântico. Skills: `language-philosophy`, `syntax-design`, `semantic-design`, `error-model-design`, `memory-model-design`, `concurrency-model-design`, `module-package-design`, `ffi-design`, `tooling-design`, `language-evolution`, `compatibility-language`, `pedagogy-language`, `standard-library-design`.

## Decision artifacts

Toda decisão de linguagem deve registrar problema, alternativas, semantic model, examples/counterexamples, ambiguity, parseability, teachability, implementation complexity, runtime cost, tooling impact, compatibility/migration e security/safety effects.

# Verification

Compiler differential testing, metamorphic/property testing, parser roundtrip, fuzzing, translation validation, interpreter-vs-compiler oracle, golden diagnostics, ABI tests, reproducible builds e semantic regression corpus.

# Tooling completo

Formatter, linter, language server/LSP, semantic highlighting, completion, refactor, debugger/DAP, docs generator, package manager, build system, test runner, REPL, playground, migration tools e editor integration.

# Design philosophies a considerar explicitamente

Simplicity/minimalism, orthogonality, zero-cost abstractions, explicitness, local reasoning, gradual disclosure, safety-by-construction, escape hatches, data-oriented design, batteries-included vs small-core, stability/evolution, tooling-first, teachability/cognitive accessibility, interoperability-first e performance predictability.

# Exit Gate

Uma proposta de linguagem não recebe nota alta apenas por sintaxe agradável. O Prumo precisa demonstrar coerência entre semantics → implementation → tooling → diagnostics → pedagogy → compatibility.