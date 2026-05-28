# dsnix

Short alias for [`mimo-reasonix`](https://www.npmjs.com/package/mimo-reasonix) — the MiMo-native coding agent.

This package is a thin shim. Installing or running `dsnix` resolves to the same `mimo-reasonix` CLI, just under a shorter command name.

## Use

```bash
# Global install
npm install -g dsnix
dsnix code my-project

# One-shot via npx
npx dsnix@latest code my-project
```

Equivalent to:

```bash
npx mimo-reasonix@latest code my-project
```

## Why a separate package?

`mimo-reasonix` is the canonical package; `dsnix` exists purely so users can type a shorter command and run `npx dsnix@latest` without typing nine letters. Version numbers track `mimo-reasonix` 1-to-1.

For docs, config, slash commands, and everything else, see the [main MiMo-Reasonix README](https://github.com/z1243161996/MiMo-Reasonix#readme).
