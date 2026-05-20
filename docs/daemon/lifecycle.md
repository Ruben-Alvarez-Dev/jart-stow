# Daemon Lifecycle

The Jart-Stow daemon (`jart-stowd`) is the heart of the system. It runs as a macOS `launchd` user agent.

## How it works

1. **Watch**: It uses the `FSEvents` API to monitor your workspace roots in real-time.
2. **Detect**: When a new directory is created or modified, it waits for a short debounce period (2s).
3. **Scan**: It performs a shallow scan (depth 3) for known artifact patterns like `node_modules`.
4. **Exclude**: It tells Time Machine and Carbon Copy Cloner to ignore these paths.
5. **Persist**: All actions are logged in the shared SQLite database.

## Control Commands

| Command | Action |
|---|---|
| `jart-stow daemon install` | Registers the agent with `launchd` |
| `jart-stow daemon start` | Starts the background process |
| `jart-stow daemon stop` | Stops the background process |
| `jart-stow daemon logs` | Tails the log file at `~/.local/share/jart-stow/daemon.log` |

## Resource Efficiency

The daemon is optimized for low resource usage:
- **WatchRootCache**: Avoids DB lookups for every filesystem event.
- **Worker Pool**: Controlled concurrency for scan jobs.
- **Debounced Processing**: Aggregates rapid changes into single batches.
