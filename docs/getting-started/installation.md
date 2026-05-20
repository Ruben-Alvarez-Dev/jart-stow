# Installation

Jart-Stow is designed to be easy to install on macOS.

## Prerequisites

- **macOS**: 12.0 (Monterey) or later.
- **Go**: 1.22 or later (for building from source).
- **Python**: 3.12 or later (for the API and TUI).

## Installation via Homebrew (Recommended)

> [!NOTE]
> Homebrew support is currently in progress.

```bash
brew tap Ruben-Alvarez-Dev/jart-stow
brew install jart-stow
```

## Manual Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/Ruben-Alvarez-Dev/jart-stow.git
   cd jart-stow
   ```

2. **Build the Go binary**:
   ```bash
   make build
   ```

3. **Set up the Python environment**:
   ```bash
   cd api
   python -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

4. **Install the Daemon**:
   ```bash
   ./jart-stow daemon install
   ./jart-stow daemon start
   ```
