# Gemini Integration

This guide explains how to compile Prumo configurations for Gemini-based workflows.

## Generic Adapter Compilation

Gemini uses the generic adapter. Run `prumo-agent compile generic --target gemini`.

## Configuration

Configure your `prumo.json` to prioritize Gemini-compatible skills (e.g., avoiding skills that rely heavily on specific non-Gemini system prompts).

## AGENTS.md Structure

For Gemini, the generated `AGENTS.md` focuses heavily on clear, step-by-step reasoning and explicit tool-use instructions tailored for Gemini's context window.
