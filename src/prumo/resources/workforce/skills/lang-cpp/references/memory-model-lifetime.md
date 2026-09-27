# Modern C++ Memory Model, Lifetimes & Ownership

## 1. Ownership Taxonomy & Semantics
- **Exclusive Ownership**: Represented strictly by `std::unique_ptr<T>`. Ownership transfer occurs exclusively via `std::move()`. Custom deleters must be declared in the template parameter or lambda wrapper.
- **Shared Ownership**: `std::shared_ptr<T>` with `std::weak_ptr<T>` to break cyclic dependencies. Must only be used when multiple decoupled subsystems genuinely share concurrent lifecycle control. Use `std::make_shared<T>()` to coalesce control block and object allocation into a single heap block.
- **Non-Owning Views**: `std::span<T>` for contiguous memory sequences and `std::string_view` for textual data. Non-owning views must **never** outlive their underlying storage.
- **Value Semantics**: Default to regular value types (`std::vector`, `std::string`, `std::array`). Value semantics eliminate pointer indirection, optimize CPU data cache locality, and simplify mental lifetime models.

---

## 2. Value Categories & Move Semantics
```
               Expressions (glvalue)
              /                     \
       lvalue                        rvalue
      (identity, no move)           /      \
                             xvalue          prvalue
                     (identity, can move)   (no identity, can move)
```
- **Moved-From Invariant**: A moved-from object is left in a "valid but unspecified" state. It can be safely destroyed or assigned to, but reading its contents or assuming specific internal values without re-initialization is undefined or bug-prone.
- **Move-Only Types**: Types managing unique OS or hardware handles (file descriptors, Vulkan handles, thread handles) must have copy operations explicitly `= delete` and move operations qualified as `noexcept` to ensure `std::vector` reallocations move rather than copy.

---

## 3. Dangling View Hazards & Iterator Invalidation
1. **Temporary Binding Hazard**:
   ```cpp
   // FATAL BUG: Temporary std::string destroyed at semicolon; view dangles!
   std::string_view sv = get_name(); // where get_name() returns std::string
   ```
2. **Container Reallocation Invalidation**:
   - `std::vector::push_back` invalidates all existing pointers, references, and `std::span` views if `capacity()` is exceeded. Pre-allocate via `reserve()`.
   - `std::unordered_map` rehash invalidates all iterators (though references to existing elements remain valid in node-based containers).

---

## 4. Concurrency & The C++ Memory Model
Modern C++ defines an explicit abstract machine memory model governed by synchronization relationships:

### 4.1 Sequenced-Before vs Happens-Before
- Modifications within a single thread are *sequenced-before*.
- Synchronization between threads requires atomic operations or lock acquisition to establish a *happens-before* edge. Data races occur when two threads concurrently access the same memory location where at least one access is a write, resulting in immediate Undefined Behavior.

### 4.2 Atomic Memory Orderings
- **`std::memory_order_relaxed`**: Guarantees atomicity of the single operation only. No synchronization or order constraints relative to surrounding memory accesses.
- **`std::memory_order_release`** (Store): Prevents all previous reads and writes from being reordered *after* this store.
- **`std::memory_order_acquire`** (Load): Prevents all subsequent reads and writes from being reordered *before* this load.
- **`std::memory_order_seq_cst`**: Sequentially consistent ordering. Provides a globally consistent total order across all atomic operations. (Default on `std::atomic<T>`).

### 4.3 Hardware False Sharing
When two threads concurrently modify independent variables that reside on the same 64-byte CPU cache line, severe bus contention and performance degradation occurs.
**Remedy**:
```cpp
struct alignas(std::hardware_destructive_interference_size) ThreadLocalCounter {
    std::atomic<std::uint64_t> count{0};
};
```
