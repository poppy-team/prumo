// High-Performance 2D Dynamic Sprite & Quad Batcher
// Demonstrating pre-allocated mapped vertex memory, static index buffer reuse,
// multi-texture slot indexing, and zero-allocation per-frame draw submission.

#include <array>
#include <cstdint>
#include <iostream>
#include <vector>

struct alignas(16) Vertex2D {
    float x, y;
    float u, v;
    uint32_t color;
    float texture_slot;
};

class SpriteBatcher2D {
public:
    static constexpr size_t MAX_QUADS = 10000;
    static constexpr size_t MAX_VERTICES = MAX_QUADS * 4;
    static constexpr size_t MAX_INDICES = MAX_QUADS * 6;
    static constexpr size_t MAX_TEXTURE_SLOTS = 16;

    SpriteBatcher2D() {
        // 1. Pre-allocate vertex storage once (persisting across entire process lifetime)
        vertex_buffer_.resize(MAX_VERTICES);

        // 2. Pre-populate static index buffer once: (0, 1, 2, 2, 3, 0) offset by 4 * quadIndex
        index_buffer_.resize(MAX_INDICES);
        for (size_t i = 0, offset = 0; i < MAX_INDICES; i += 6, offset += 4) {
            index_buffer_[i + 0] = static_cast<uint32_t>(offset + 0);
            index_buffer_[i + 1] = static_cast<uint32_t>(offset + 1);
            index_buffer_[i + 2] = static_cast<uint32_t>(offset + 2);
            index_buffer_[i + 3] = static_cast<uint32_t>(offset + 2);
            index_buffer_[i + 4] = static_cast<uint32_t>(offset + 3);
            index_buffer_[i + 5] = static_cast<uint32_t>(offset + 0);
        }

        std::cout << "[INIT] SpriteBatcher pre-allocated " << MAX_VERTICES
                  << " vertices and " << MAX_INDICES << " static indices.\n";
    }

    void begin() noexcept {
        quad_count_ = 0;
        texture_slot_count_ = 0;
        draw_call_count_ = 0;
    }

    void draw_sprite(float x, float y, float w, float h,
                     float u0, float v0, float u1, float v1,
                     uint32_t color_rgba, uint32_t texture_id) {
        // Resolve texture slot index
        float texture_slot = -1.0f;
        for (size_t i = 0; i < texture_slot_count_; ++i) {
            if (bound_textures_[i] == texture_id) {
                texture_slot = static_cast<float>(i);
                break;
            }
        }

        // If texture not currently bound, bind it or trigger flush if slots exhausted
        if (texture_slot < 0.0f) {
            if (texture_slot_count_ >= MAX_TEXTURE_SLOTS) {
                flush();
            }
            texture_slot = static_cast<float>(texture_slot_count_);
            bound_textures_[texture_slot_count_++] = texture_id;
        }

        // If vertex buffer is full, flush to GPU
        if (quad_count_ >= MAX_QUADS) {
            flush();
        }

        // Direct write to pre-allocated buffer (Zero dynamic allocation)
        size_t v_idx = quad_count_ * 4;
        vertex_buffer_[v_idx + 0] = {x,     y,     u0, v0, color_rgba, texture_slot};
        vertex_buffer_[v_idx + 1] = {x + w, y,     u1, v0, color_rgba, texture_slot};
        vertex_buffer_[v_idx + 2] = {x + w, y + h, u1, v1, color_rgba, texture_slot};
        vertex_buffer_[v_idx + 3] = {x,     y + h, u0, v1, color_rgba, texture_slot};

        quad_count_++;
    }

    void flush() {
        if (quad_count_ == 0) return;

        draw_call_count_++;
        // In real backend:
        // 1. Upload vertex_buffer_.data() range [0 .. quad_count_ * 4] to GPU ring buffer
        // 2. Bind bound_textures_ array [0 .. texture_slot_count_] to descriptor set
        // 3. Issue DrawIndexed(quad_count_ * 6)

        // Reset batch accumulator for next batch
        quad_count_ = 0;
        texture_slot_count_ = 0;
    }

    void end() {
        flush();
    }

    [[nodiscard]] size_t total_draw_calls() const noexcept { return draw_call_count_; }

private:
    std::vector<Vertex2D> vertex_buffer_;
    std::vector<uint32_t> index_buffer_;
    std::array<uint32_t, MAX_TEXTURE_SLOTS> bound_textures_{};
    size_t quad_count_{0};
    size_t texture_slot_count_{0};
    size_t draw_call_count_{0};
};

int main() {
    std::cout << "=== High-Performance 2D Sprite Batching Demonstration ===\n";

    SpriteBatcher2D batcher;
    batcher.begin();

    // Render 50,000 sprites across 4 distinct textures
    for (int i = 0; i < 50000; ++i) {
        uint32_t tex_id = static_cast<uint32_t>((i % 4) + 100);
        batcher.draw_sprite(
            static_cast<float>(i % 800), static_cast<float>(i / 800),
            16.0f, 16.0f,
            0.0f, 0.0f, 1.0f, 1.0f,
            0xFFFFFFFF, tex_id
        );
    }

    batcher.end();

    std::cout << "Rendered 50,000 sprites in " << batcher.total_draw_calls()
              << " GPU draw calls (Capacity per batch: 10,000 quads).\n";
    return 0;
}
