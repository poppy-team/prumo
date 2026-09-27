package main

import "core:fmt"
import "core:mem"

// Strong domain types to prevent accidental type confusion
Entity_Id :: distinct u64
Health    :: distinct f32

// Particle component layout suitable for SIMD and CPU cache lines
Particle :: struct {
    pos:   [3]f32,
    vel:   [3]f32,
    life:  Health,
    id:    Entity_Id,
}

// Simulated error set
Simulation_Error :: enum {
    None,
    Invalid_Capacity,
    Buffer_Overflow,
}

// Procedure demonstrating error return discipline
spawn_particles :: proc(count: int, allocator := context.allocator) -> (#soa[dynamic]Particle, Simulation_Error) {
    if count <= 0 {
        return {}, .Invalid_Capacity
    }

    particles := make(#soa[dynamic]Particle, allocator)

    for i in 0..<count {
        p := Particle{
            pos  = [3]f32{f32(i), 0.0, 0.0},
            vel  = [3]f32{0.0, 1.0, 0.0},
            life = Health(100.0),
            id   = Entity_Id(u64(i + 1)),
        }
        append_soa(&particles, p)
    }

    return particles, .None
}

main :: proc() {
    // 1. Initialize tracking allocator to detect leaks and bad frees
    track: mem.Tracking_Allocator
    mem.tracking_allocator_init(&track, context.allocator)
    defer mem.tracking_allocator_destroy(&track)

    // Set tracking allocator as primary context allocator
    context.allocator = mem.tracking_allocator(&track)

    fmt.println("[Prumo Odin] Running memory and #soa demonstration...")

    {
        // 2. Spawn SOA collection
        particles, err := spawn_particles(128, context.allocator)
        if err != .None {
            fmt.eprintln("Failed to spawn particles:", err)
            return
        }
        defer delete_soa(particles)

        // 3. Cache-coherent update: iterate exclusively across life values
        dt: Health = 1.5
        for &life in particles.life {
            life -= dt
        }

        fmt.printf("Successfully updated %d particles in SOA memory layout.\n", len(particles))
    }

    // 4. Validate zero memory leaks
    if len(track.allocation_map) > 0 {
        fmt.eprintf("ERROR: %d memory leaks detected!\n", len(track.allocation_map))
        for _, leak in track.allocation_map {
            fmt.eprintf("  -> %v leaked %m bytes\n", leak.location, leak.size)
        }
    } else {
        fmt.println("[Prumo Odin] Tracking Allocator: 0 leaks, 0 bad frees. All memory reclaimed cleanly.")
    }
}
