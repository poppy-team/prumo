module safe_example;

import std::io;
import std::mem;

// Distinct domain primitives
distinct EntityId = u64;
distinct Timestamp = u64;

// Explicit fault enumeration
fault BufferError {
    EMPTY_SLICE,
    INDEX_OUT_OF_BOUNDS,
    ALLOCATION_FAILURE
}

/**
 * Pure calculation function asserting mathematical constraints
 */
fn int calculate_checksum(int[] data) @pure {
    int sum = 0;
    foreach (val : data) {
        sum = sum + val;
    }
    return sum;
}

/**
 * Demonstrates bounded slices, fault returns, and defer rollbacks
 */
fn int! process_and_aggregate(int[] items, EntityId entity_id) @nodiscard {
    if (items.len == 0) {
        return BufferError.EMPTY_SLICE?;
    }

    // Allocate temporary scratch working buffer
    int* scratch = (int*)mem::alloc(items.len * @size_of(int));
    if (scratch == null) {
        return BufferError.ALLOCATION_FAILURE?;
    }

    // Standard defer: guaranteed execution at scope exit
    defer io::printfln("[C3] Reclaiming scratch buffer for entity %d...", (u64)entity_id);

    // Conditional rollback: execute free ONLY if an error occurs
    defer (catch err) {
        io::printfln("[C3] Error detected (%s), rolling back allocated memory.", err);
        mem::free(scratch);
    }

    // No-catch defer: execute cleanup and finalization on successful completion
    defer (no_catch) {
        io::printfln("[C3] Normal completion, cleaning up scratch memory.");
        mem::free(scratch);
    }

    // Copy and process slice safely
    for (usz i = 0; i < items.len; i++) {
        scratch[i] = items[i] * 2;
    }

    int total = calculate_checksum(items);
    return total;
}

fn int main() {
    io::printfln("=== Prumo C3 Safety Demonstration ===");

    int[4] source_data = { 10, 20, 30, 40 };
    int[] slice_view = &source_data;

    EntityId test_entity = (EntityId)101;

    // Execute with explicit catch handling
    if (catch err = process_and_aggregate(slice_view, test_entity)) {
        io::printfln("Processing failed with fault: %s", err);
        return 1;
    }

    io::printfln("Processing succeeded cleanly.");
    return 0;
}
