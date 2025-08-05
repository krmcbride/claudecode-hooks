# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go project that provides hooks for Claude Code. See: @docs/claude-code-hooks.md

## Essential Commands

### Building and Development

- `make build` - Build all hook binaries to `build/` directory
- `make dev` - Run full development workflow (tools setup, format, lint, test, build)
- `make test` - Run all tests
- `make check` - Run format, lint, and test in sequence

### Individual Hook Management

- `make install` - Install hooks to project `.claude/hooks/` directory
- `make install-user` - Install hooks to user `~/.claude/hooks/` directory

### Build System

The project uses a modular Makefile system:

- `Makefile` - Main configuration and help
- `makefiles/build.mk` - Build, install, uninstall targets
- `makefiles/dev.mk` - Development workflow targets
- `makefiles/tools.mk` - Tool installation and management

