const std = @import("std");

/// Safe Dynamic Buffer demonstrating explicit allocation,
/// errdefer rollback semantics, and leak-free destruction.
pub const SafeBuffer = struct {
    allocator: std.mem.Allocator,
    items: []u8,
    len: usize,

    pub fn init(allocator: std.mem.Allocator, initial_capacity: usize) !SafeBuffer {
        if (initial_capacity == 0) return error.InvalidCapacity;

        const memory = try allocator.alloc(u8, initial_capacity);
        errdefer allocator.free(memory);

        @memset(memory, 0);

        return SafeBuffer{
            .allocator = allocator,
            .items = memory,
            .len = 0,
        };
    }

    pub fn deinit(self: *SafeBuffer) void {
        self.allocator.free(self.items);
        self.* = undefined;
    }

    pub fn append(self: *SafeBuffer, byte: u8) !void {
        if (self.len >= self.items.len) {
            const new_capacity = self.items.len * 2;
            const new_memory = try self.allocator.realloc(self.items, new_capacity);
            self.items = new_memory;
        }

        self.items[self.len] = byte;
        self.len += 1;
    }

    pub fn slice(self: *const SafeBuffer) []const u8 {
        return self.items[0..self.len];
    }
};

/// Generic Bounded Stack with compile-time invariant checks
pub fn BoundedStack(comptime T: type, comptime capacity: usize) type {
    comptime {
        if (capacity == 0) {
            @compileError("BoundedStack capacity must be greater than 0");
        }
    }

    return struct {
        const Self = @This();
        buffer: [capacity]T = undefined,
        count: usize = 0,

        pub fn push(self: *Self, item: T) !void {
            if (self.count >= capacity) return error.StackOverflow;
            self.buffer[self.count] = item;
            self.count += 1;
        }

        pub fn pop(self: *Self) !T {
            if (self.count == 0) return error.StackUnderflow;
            self.count -= 1;
            return self.buffer[self.count];
        }

        pub fn isEmpty(self: *const Self) bool {
            return self.count == 0;
        }
    };
}

test "SafeBuffer allocation and reallocation leak test" {
    const allocator = std.testing.allocator;

    var buf = try SafeBuffer.init(allocator, 4);
    defer buf.deinit();

    try buf.append(10);
    try buf.append(20);
    try buf.append(30);
    try buf.append(40);
    // Triggers realloc
    try buf.append(50);

    try std.testing.expectEqual(buf.len, 5);
    try std.testing.expectEqual(buf.slice()[0], 10);
    try std.testing.expectEqual(buf.slice()[4], 50);
}

test "BoundedStack comptime and stack allocation" {
    var stack = BoundedStack(u32, 8){};
    try stack.push(42);
    try stack.push(99);

    const val = try stack.pop();
    try std.testing.expectEqual(val, 99);
    try std.testing.expect(!stack.isEmpty());
}

test "ArenaAllocator batch lifecycle" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    const arena_alloc = arena.allocator();
    const str1 = try arena_alloc.dupe(u8, "Prumo");
    const str2 = try arena_alloc.dupe(u8, "Zig");

    try std.testing.expectEqualStrings(str1, "Prumo");
    try std.testing.expectEqualStrings(str2, "Zig");
}
