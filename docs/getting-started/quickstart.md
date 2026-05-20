# Quick Start

Get up and running with Jart-Stow in minutes.

## 1. Configure Workspace Roots

Tell Jart-Stow which folders to monitor for development projects:

```bash
jart-stow watch-root add ~/Code
```

## 2. Start the Daemon

Ensure the background process is running to monitor file changes:

```bash
jart-stow daemon start
```

## 3. Launch the TUI

Experience the full power of Jart-Stow with the interactive terminal interface:

```bash
make tui
```

## 4. Review Junk

Navigate to the **Hygiene** screen in the TUI to see unused Docker images, APFS snapshots, and system caches that you can safely clean.

## 5. Verify Exclusions

Check your Time Machine settings or use the `tmutil` command to verify that `node_modules` and other artifacts are being excluded:

```bash
tmutil isexcluded ~/Code/my-project/node_modules
```
