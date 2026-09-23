#!/usr/bin/env python3
"""
Production-Grade AST Transformation and Codemod Demonstration
Features:
- Subclasses Python's ast.NodeTransformer
- Modernizes legacy 'os.path.join(a, b)' calls to pathlib 'Path(a) / b'
- Injects necessary imports hygienically without duplicate imports
- Guarantees strict idempotence: T(T(code)) == T(code)
- Full self-test verification suite
"""

import ast
import sys
from typing import Optional


class PathlibMigrationTransformer(ast.NodeTransformer):
    """
    Transforms `os.path.join(x, y, ...)` into `Path(x) / y / ...`
    and ensures `from pathlib import Path` is imported.
    """

    def __init__(self):
        super().__init__()
        self.modified = False
        self.has_pathlib_import = False

    def visit_ImportFrom(self, node: ast.ImportFrom) -> ast.AST:
        if node.module == "pathlib":
            for alias in node.names:
                if alias.name == "Path":
                    self.has_pathlib_import = True
        return self.generic_visit(node)

    def visit_Call(self, node: ast.Call) -> ast.AST:
        # First recurse down into children
        self.generic_visit(node)

        # Match pattern: os.path.join(...)
        if isinstance(node.func, ast.Attribute):
            if (
                isinstance(node.func.value, ast.Attribute)
                and node.func.value.attr == "path"
                and isinstance(node.func.value.value, ast.Name)
                and node.func.value.value.id == "os"
                and node.func.attr == "join"
            ):
                if not node.args:
                    return node

                self.modified = True

                # Build Path(args[0])
                first_arg = node.args[0]
                path_call = ast.Call(
                    func=ast.Name(id="Path", ctx=ast.Load()),
                    args=[first_arg],
                    keywords=[],
                )
                ast.copy_location(path_call, first_arg)

                # Chain remaining arguments using BinOp with Div (/)
                current_node = path_call
                for next_arg in node.args[1:]:
                    current_node = ast.BinOp(
                        left=current_node,
                        op=ast.Div(),
                        right=next_arg,
                    )
                    ast.copy_location(current_node, next_arg)

                return current_node

        return node

    def transform_module(self, tree: ast.Module) -> ast.Module:
        transformed = self.visit(tree)
        ast.fix_missing_locations(transformed)

        # If modified and Path import was missing, inject it at top
        if self.modified and not self.has_pathlib_import:
            import_node = ast.ImportFrom(
                module="pathlib",
                names=[ast.alias(name="Path", asname=None)],
                level=0,
            )
            transformed.body.insert(0, import_node)
            ast.fix_missing_locations(transformed)

        return transformed


def apply_codemod(source_code: str) -> str:
    tree = ast.parse(source_code)
    transformer = PathlibMigrationTransformer()
    new_tree = transformer.transform_module(tree)
    return ast.unparse(new_tree)


def run_self_test():
    print("[AST Transformer] Starting self-test and idempotency verification...")

    sample_input = """import os

def build_data_path(base_dir, filename):
    return os.path.join(base_dir, "data", filename)
"""

    # 1. First Transformation Pass: T(S0) -> S1
    s1 = apply_codemod(sample_input)
    print("Pass 1 Result:\n" + s1 + "\n" + "-" * 40)

    assert "from pathlib import Path" in s1, "Expected Path import to be injected"
    assert "Path(base_dir) / 'data' / filename" in s1 or 'Path(base_dir) / "data" / filename' in s1, (
        f"Unexpected transform output: {s1}"
    )

    # 2. Second Transformation Pass (Idempotency Test): T(S1) -> S2
    s2 = apply_codemod(s1)
    assert s1 == s2, (
        f"CRITICAL: Non-idempotent transformation detected!\nS1:\n{s1}\n\nS2:\n{s2}"
    )
    print("Pass 2 Idempotency Verified: T(T(x)) == T(x)")

    # 3. Third Test: Unrelated code is untouched
    unrelated_input = """def add(a, b):
    return a + b
"""
    unrelated_out = apply_codemod(unrelated_input)
    assert "from pathlib import Path" not in unrelated_out, "Should not inject import when unneeded"
    print("Untouched code correctly remains unaltered.")

    print("[AST Transformer] All verification assertions passed successfully!")


if __name__ == "__main__":
    run_self_test()
