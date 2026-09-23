// Modern C++20 Idiomatic RAII, Concepts, and Span Safety Reference
// Demonstrating the Rule of Zero/Five, concepts constraints, and span parameterization.

#include <array>
#include <concepts>
#include <cstdint>
#include <iostream>
#include <memory>
#include <span>
#include <string>
#include <string_view>
#include <vector>

// 1. C++20 Concept constraining printable elements
template <typename T>
concept Printable = requires(T a) {
    { std::cout << a } -> std::same_as<std::ostream&>;
};

// 2. Custom Move-Only Resource Handle (Rule of Five)
class ManagedResource {
public:
    explicit ManagedResource(std::string_view name)
        : name_(name), handle_id_(1001) {
        std::cout << "[ACQUIRE] " << name_ << " (id=" << handle_id_ << ")\n";
    }

    ~ManagedResource() noexcept {
        if (handle_id_ != 0) {
            std::cout << "[RELEASE] " << name_ << " (id=" << handle_id_ << ")\n";
        }
    }

    // Move-only semantics: prevent accidental shallow copy of unique resource
    ManagedResource(const ManagedResource&) = delete;
    ManagedResource& operator=(const ManagedResource&) = delete;

    ManagedResource(ManagedResource&& other) noexcept
        : name_(std::move(other.name_)), handle_id_(other.handle_id_) {
        other.handle_id_ = 0; // Invalidate moved-from resource
    }

    ManagedResource& operator=(ManagedResource&& other) noexcept {
        if (this != &other) {
            name_ = std::move(other.name_);
            handle_id_ = other.handle_id_;
            other.handle_id_ = 0;
        }
        return *this;
    }

    // 3. Constrained generic processor using std::span for boundary safety across any extent
    template <Printable T, std::size_t Extent = std::dynamic_extent>
    void process_batch(std::span<const T, Extent> items) const {
        std::cout << "Processing " << items.size() << " items for " << name_ << ":\n";
        for (const auto& item : items) {
            std::cout << "  - Item: " << item << '\n';
        }
    }

    [[nodiscard]] std::string_view name() const noexcept { return name_; }
    [[nodiscard]] bool is_valid() const noexcept { return handle_id_ != 0; }

private:
    std::string name_;
    std::uint64_t handle_id_{0};
};

int main() {
    std::cout << "=== Modern C++20 RAII & Safety Demonstration ===\n";

    // Managed via smart pointer (Rule of Zero at callsite)
    auto resource = std::make_unique<ManagedResource>("ComputeContext");

    // Pass data by non-owning view std::span
    const std::vector<std::int32_t> data_vector = {42, 84, 126, 168};
    resource->process_batch(std::span<const std::int32_t>{data_vector});

    const std::array<std::string_view, 2> string_array = {"alpha", "omega"};
    resource->process_batch(std::span<const std::string_view>{string_array});

    // Test move semantics
    ManagedResource moved_res = std::move(*resource);
    std::cout << "Moved resource is valid: " << std::boolalpha << moved_res.is_valid() << '\n';

    return 0;
}
