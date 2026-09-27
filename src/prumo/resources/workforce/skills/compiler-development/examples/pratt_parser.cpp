/**
 * Pratt Parser (Top-Down Operator Precedence) Demonstration
 * Features:
 * - Dynamic token stream with source positions
 * - Pratt binding power algorithm handling infix, prefix, and right-associative operators
 * - AST generation and expression evaluator
 * - Verified mathematical precedence: 2 + 3 * 4 == 14, 2 ^ 3 ^ 2 == 512
 */

#include <iostream>
#include <vector>
#include <string>
#include <string_view>
#include <memory>
#include <cmath>
#include <cassert>
#include <sstream>

enum class TokenType {
    Number,
    Plus,
    Minus,
    Star,
    Slash,
    Caret, // Exponentiation (right-associative)
    LParen,
    RParen,
    Eof
};

struct Token {
    TokenType type;
    double value{0.0};
    size_t pos{0};
};

class Lexer {
public:
    explicit Lexer(std::string_view source) : src_(source), index_(0) {}

    Token next_token() {
        skip_whitespace();
        if (index_ >= src_.size()) {
            return Token{TokenType::Eof, 0.0, index_};
        }

        char c = src_[index_];
        size_t start = index_++;

        switch (c) {
            case '+': return Token{TokenType::Plus, 0.0, start};
            case '-': return Token{TokenType::Minus, 0.0, start};
            case '*': return Token{TokenType::Star, 0.0, start};
            case '/': return Token{TokenType::Slash, 0.0, start};
            case '^': return Token{TokenType::Caret, 0.0, start};
            case '(': return Token{TokenType::LParen, 0.0, start};
            case ')': return Token{TokenType::RParen, 0.0, start};
            default:
                if (std::isdigit(c) || c == '.') {
                    index_ = start;
                    return parse_number();
                }
                throw std::runtime_error("Unexpected character in input");
        }
    }

private:
    std::string_view src_;
    size_t index_;

    void skip_whitespace() {
        while (index_ < src_.size() && std::isspace(src_[index_])) {
            index_++;
        }
    }

    Token parse_number() {
        size_t start = index_;
        while (index_ < src_.size() && (std::isdigit(src_[index_]) || src_[index_] == '.')) {
            index_++;
        }
        std::string num_str(src_.substr(start, index_ - start));
        double val = std::stod(num_str);
        return Token{TokenType::Number, val, start};
    }
};

// Abstract Syntax Tree (AST) Nodes
struct Expr {
    virtual ~Expr() = default;
    virtual double evaluate() const = 0;
    virtual std::string to_string() const = 0;
};

struct NumberExpr : public Expr {
    double value;
    explicit NumberExpr(double val) : value(val) {}
    double evaluate() const override { return value; }
    std::string to_string() const override {
        std::ostringstream ss;
        ss << value;
        return ss.str();
    }
};

struct UnaryExpr : public Expr {
    TokenType op;
    std::unique_ptr<Expr> operand;
    UnaryExpr(TokenType o, std::unique_ptr<Expr> rhs) : op(o), operand(std::move(rhs)) {}
    double evaluate() const override {
        double val = operand->evaluate();
        return (op == TokenType::Minus) ? -val : val;
    }
    std::string to_string() const override {
        return "(-" + operand->to_string() + ")";
    }
};

struct BinaryExpr : public Expr {
    TokenType op;
    std::unique_ptr<Expr> left;
    std::unique_ptr<Expr> right;

    BinaryExpr(TokenType o, std::unique_ptr<Expr> l, std::unique_ptr<Expr> r)
        : op(o), left(std::move(l)), right(std::move(r)) {}

    double evaluate() const override {
        double l = left->evaluate();
        double r = right->evaluate();
        switch (op) {
            case TokenType::Plus: return l + r;
            case TokenType::Minus: return l - r;
            case TokenType::Star: return l * r;
            case TokenType::Slash:
                if (r == 0.0) throw std::runtime_error("Division by zero");
                return l / r;
            case TokenType::Caret: return std::pow(l, r);
            default: throw std::runtime_error("Unknown binary operator");
        }
    }

    std::string to_string() const override {
        char op_char = '?';
        switch (op) {
            case TokenType::Plus: op_char = '+'; break;
            case TokenType::Minus: op_char = '-'; break;
            case TokenType::Star: op_char = '*'; break;
            case TokenType::Slash: op_char = '/'; break;
            case TokenType::Caret: op_char = '^'; break;
            default: break;
        }
        return "(" + left->to_string() + " " + op_char + " " + right->to_string() + ")";
    }
};

class PrattParser {
public:
    explicit PrattParser(std::string_view source) : lexer_(source) {
        advance();
    }

    std::unique_ptr<Expr> parse_expression(int min_bp = 0) {
        Token token = current_;
        advance();

        // 1. Prefix parsing (NUD)
        std::unique_ptr<Expr> left;
        if (token.type == TokenType::Number) {
            left = std::make_unique<NumberExpr>(token.value);
        } else if (token.type == TokenType::Minus) {
            // Unary prefix minus has high binding power (70)
            auto operand = parse_expression(70);
            left = std::make_unique<UnaryExpr>(TokenType::Minus, std::move(operand));
        } else if (token.type == TokenType::LParen) {
            left = parse_expression(0);
            consume(TokenType::RParen, "Expected ')' closing parenthesis");
        } else {
            throw std::runtime_error("Unexpected token in prefix position");
        }

        // 2. Infix parsing loop (LED)
        while (true) {
            auto [lbp, rbp] = get_infix_binding_power(current_.type);
            if (lbp <= min_bp) break;

            Token op_token = current_;
            advance();

            auto right = parse_expression(rbp);
            left = std::make_unique<BinaryExpr>(op_token.type, std::move(left), std::move(right));
        }

        return left;
    }

private:
    Lexer lexer_;
    Token current_;

    void advance() {
        current_ = lexer_.next_token();
    }

    void consume(TokenType type, std::string_view err_msg) {
        if (current_.type != type) {
            throw std::runtime_error(std::string(err_msg));
        }
        advance();
    }

    // Binding power pairs (Left Binding Power, Right Binding Power)
    std::pair<int, int> get_infix_binding_power(TokenType op) {
        switch (op) {
            case TokenType::Plus:
            case TokenType::Minus:
                return {10, 11}; // Left-associative: LBP < RBP
            case TokenType::Star:
            case TokenType::Slash:
                return {20, 21}; // Left-associative
            case TokenType::Caret:
                return {31, 30}; // Right-associative: LBP > RBP (2 ^ 3 ^ 2 == 2 ^ (3 ^ 2))
            default:
                return {0, 0};
        }
    }
};

int main() {
    std::cout << "[PrattParser] Running mathematical grammar verification...\n";

    // Test 1: Standard precedence (2 + 3 * 4 == 14)
    {
        PrattParser p("2 + 3 * 4");
        auto ast = p.parse_expression();
        assert(ast->evaluate() == 14.0);
        std::cout << "  ✓ '2 + 3 * 4' -> " << ast->to_string() << " == " << ast->evaluate() << "\n";
    }

    // Test 2: Parentheses override ((2 + 3) * 4 == 20)
    {
        PrattParser p("(2 + 3) * 4");
        auto ast = p.parse_expression();
        assert(ast->evaluate() == 20.0);
        std::cout << "  ✓ '(2 + 3) * 4' -> " << ast->to_string() << " == " << ast->evaluate() << "\n";
    }

    // Test 3: Right-associative exponentiation (2 ^ 3 ^ 2 == 512, not 64)
    {
        PrattParser p("2 ^ 3 ^ 2");
        auto ast = p.parse_expression();
        assert(ast->evaluate() == 512.0);
        std::cout << "  ✓ '2 ^ 3 ^ 2' -> " << ast->to_string() << " == " << ast->evaluate() << "\n";
    }

    // Test 4: Unary prefix negation (-5 + 10 == 5)
    {
        PrattParser p("-5 + 10");
        auto ast = p.parse_expression();
        assert(ast->evaluate() == 5.0);
        std::cout << "  ✓ '-5 + 10' -> " << ast->to_string() << " == " << ast->evaluate() << "\n";
    }

    std::cout << "[PrattParser] All precedence and associativity assertions passed!\n";
    return 0;
}
