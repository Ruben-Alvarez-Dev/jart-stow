# Jart-Stow TUI Specification

**Version:** 1.0.0  
**Status:** Draft  
**Author:** Ruben Alvarez  
**Date:** 2026-05-05  

---

## 1. Overview

The Jart-Stow TUI is built with the **Charmbracelet ecosystem**:

| Library | Purpose |
|---|---|
| [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Elm-like TUI framework |
| [Bubbles](https://github.com/charmbracelet/bubbles) | Reusable components (table, viewport, spinner, textinput) |
| [Lip Gloss](https://github.com/charmbracelet/lipgloss) | Declarative terminal styling |
| [Harmonica](https://github.com/charmbracelet/harmonica) | Spring animations (transitions) |
| [Glamour](https://github.com/charmbracelet/glamour) | Markdown rendering (help/report screens) |

---

## 2. Screen Map

```
                    ┌─────────────────┐
                    │    DASHBOARD     │  ← Home screen
                    │  (default view)  │
                    └───┬────┬────┬────┘
                        │    │    │
          ┌─────────────┘    │    └─────────────┐
          ▼                  ▼                  ▼
┌─────────────────┐ ┌──────────────┐ ┌─────────────────┐
│    SCANNER      │ │  EXCLUSIONS  │ │     RULES       │
│  (scan volumes) │ │  (view/edit) │ │  (manage rules) │
└────────┬────────┘ └──────────────┘ └────────┬────────┘
         │                                     │
         ▼                                     ▼
┌─────────────────┐                   ┌─────────────────┐
│     AUDIT       │                   │    HYGIENE      │
│  (workspace     │                   │  (system junk)  │
│   inspection)   │                   └────────┬────────┘
└─────────────────┘                            │
                                               ▼
                                      ┌─────────────────┐
                                      │     REPORT      │
                                      │  (stats &       │
                                      │   history)      │
                                      └─────────────────┘
```

### Navigation

| Key | Action |
|---|---|
| `1` | Dashboard |
| `2` | Scanner |
| `3` | Exclusions |
| `4` | Rules |
| `5` | Audit |
| `6` | Hygiene |
| `7` | Report |
| `Tab` | Cycle focus between panels |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

---

## 3. Screen Specifications

### 3.1 Dashboard — Home Screen

```
┌──────────────────────────────────────────────────────────┐
│  JART-STOW                               Ruben's Edition │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌─── System Status ──────────┐  ┌── Quick Stats ──────┐│
│  │  Daemon:    ● RUNNING       │  │  Projects:    47    ││
│  │  Watched:   /Users/ruben/Code│  │  Exclusions:  312   ││
│  │  Time Machine:  ✓ Active    │  │  Space Saved: 8.4GB ││
│  │  CCC:           ✓ Active    │  │  Last Scan:  2m ago ││
│  └─────────────────────────────┘  └────────────────────┘│
│                                                          │
│  ┌── Recent Activity ──────────────────────────────────┐│
│  │  20:30:15  ✓  new-project      4 folders excluded   ││
│  │  20:28:02  ✓  CAAL             node_modules (1.2GB) ││
│  │  20:15:00  ⚠  Jart-OS          .venv over threshold ││
│  │  20:10:33  ✓  Voice-chat       2 new exclusions     ││
│  │  20:01:00  ●  Daemon started                        ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌── Top Offenders (by size) ──────────────────────────┐│
│  │  1. CAAL/node_modules          2.3 GB               ││
│  │  2. BrowserOS/.next             1.8 GB               ││
│  │  3. AUTOCLAW/.venv              950 MB               ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  1-6 Screens  │  ? Help  │  q Quit                       │
└──────────────────────────────────────────────────────────┘
```

**Layout:** 3 rows × 2 columns (top) + 2 full-width panels (bottom)  
**Components:** `StatusCard`, `QuickStatsCard`, `ActivityLog`, `TopOffendersTable`

---

### 3.2 Scanner — Volume/Project Scanning

```
┌──────────────────────────────────────────────────────────┐
│  SCANNER                                     [Scan Mode] │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌── Volumes & Paths ──────────┐  ┌── Scan Results ────┐│
│  │                              │  │                     ││
│  │  [x] 🏠 Home                 │  │  ⠋ Scanning...      ││
│  │  [ ] 💿 Macintosh HD         │  │                     ││
│  │  [x] 📁 /Code                │  │  node_modules (23)  ││
│  │  [ ] 💿 BackupDrive          │  │  .venv (8)          ││
│  │                              │  │  build (4)          ││
│  │  ──────────────────────      │  │  .next (3)          ││
│  │  Patterns:                   │  │                     ││
│  │  [x] node_modules            │  │                     ││
│  │  [x] .venv                   │  │                     ││
│  │  [x] target                  │  │                     ││
│  │  [x] build                   │  │                     ││
│  │  [+] Add pattern...          │  │                     ││
│  └──────────────────────────────┘  └────────────────────┘│
│                                                          │
│  ┌── Scan Log ─────────────────────────────────────────┐│
│  │  ✓ /Code/CAAL: 12 folders found (2.8 GB)           ││
│  │  ✓ /Code/Jart-OS: 3 folders found (450 MB)          ││
│  │  ⠋ Scanning /Code/BrowserOS...                      ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  Space: Toggle  │  Enter: Scan  │  Tab: Switch panels     │
└──────────────────────────────────────────────────────────┘
```

**Components:** `VolumeTree`, `PatternChecklist`, `ScanResultsTree`, `ScanLog`

---

### 3.3 Exclusions — View & Manage

```
┌──────────────────────────────────────────────────────────┐
│  EXCLUSIONS                             312 active        │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  Filters: [TM] [CCC] [All]  │  Sort: [Project] [Size] [Date] │
│                                                          │
│  ┌── Exclusion List ───────────────────────────────────┐│
│  │  #  │ Project     │ Path              │ Size   │ Sys ││
│  │─────┼─────────────┼───────────────────┼────────┼─────││
│  │  1  │ CAAL        │ .../node_modules  │ 2.3 GB │ TM+ ││
│  │  2  │ CAAL        │ .../.venv          │ 850 MB │ TM+ ││
│  │  3  │ BrowserOS   │ .../.next          │ 1.8 GB │ CCC ││
│  │  4  │ Jart-OS     │ .../target         │ 200 MB │ TM  ││
│  │  5  │ AUTOCLAW    │ .../.venv          │ 950 MB │ TM+ ││
│  │  ⋮  │     ⋮       │        ⋮           │   ⋮    │ ⋮   ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌── Selected Exclusion ───────────────────────────────┐│
│  │  Project:    CAAL                                    ││
│  │  Path:       /Users/ruben/Code/CAAL/node_modules     ││
│  │  Size:       2.3 GB                                  ││
│  │  Excluded:   2026-05-01 14:30 (TM + CCC)             ││
│  │                                                      ││
│  │  [Restore] [Restore All in Project]                  ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ↑↓ Navigate  │  Enter: Select  │  R: Restore  │  F: Filter │
└──────────────────────────────────────────────────────────┘
```

**Components:** `FilterBar`, `ExclusionTable` (sortable), `ExclusionDetail`

---

### 3.4 Rules — Hygiene Rule Manager

```
┌──────────────────────────────────────────────────────────┐
│  RULES                                         8 active   │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌── Global Defaults ──────────────────────────────────┐│
│  │  Pattern        │ Max Size  │ Action   │ Enabled    ││
│  │─────────────────┼───────────┼──────────┼────────────││
│  │  node_modules   │ 500 MB    │ exclude  │ ✓          ││
│  │  .venv          │ 500 MB    │ exclude  │ ✓          ││
│  │  __pycache__    │ 100 MB    │ exclude  │ ✓          ││
│  │  .next          │ 200 MB    │ warn     │ ✓          ││
│  │  build          │ 300 MB    │ alert    │ ✓          ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌── Per-Project Overrides ────────────────────────────┐│
│  │  Project  │ Pattern        │ Max Size  │ Action     ││
│  │───────────┼────────────────┼───────────┼────────────││
│  │  CAAL     │ .venv          │ 1 GB      │ warn       ││
│  │  Voice-chat│ audio_cache   │ 100 MB    │ clean      ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌── Rule Editor ──────────────────────────────────────┐│
│  │  Project:   [Global ▼]     Pattern: [node_modules  ]││
│  │  Max Size:  [500    ] MB   Action:  [exclude    ▼]  ││
│  │  Enabled:   [✓]                                      ││
│  │                                                      ││
│  │           [Save]  [Delete]  [Cancel]                 ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  A: Add Rule  │  E: Edit  │  D: Delete  │  ↑↓: Navigate  │
└──────────────────────────────────────────────────────────┘
```

**Components:** `RuleTable` (×2), `RuleEditor` (textinputs, dropdowns)

---

### 3.5 Audit — Workspace Inspection

```
┌──────────────────────────────────────────────────────────┐
│  AUDIT                                 /Users/ruben/Code  │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌── Project Tree ────────────┐  ┌── Project Detail ───┐│
│  │                             │  │                      ││
│  │  📁 /Code                   │  │  CAAL                ││
│  │  ├─ 📁 CAAL        ⚠ 4.1G  │  │  Status: ⚠ Warning   ││
│  │  ├─ 📁 Jart-OS     ✓ 0M    │  │                      ││
│  │  ├─ 📁 BrowserOS   ⚠ 2.3G  │  │  node_modules  2.3G  ││
│  │  ├─ 📁 AUTOCLAW    ⚠ 1.1G  │  │  .venv         850M  ││
│  │  ├─ 📁 Voice-chat  ✓ 0M    │  │  audio_cache   120M ⚠││
│  │  ├─ 📁 CLIs        ✓ 0M    │  │  build         340M  ││
│  │  ├─ 📁 MCP-servers ⚠ 500M  │  │  ───────────── 4.1G  ││
│  │  └─ ...                    │  │                      ││
│  │                             │  │  Rules violated:     ││
│  │                             │  │  • node_modules >    ││
│  │  ──────────────────────     │  │    500MB threshold   ││
│  │  Legend:                    │  │  • audio_cache >     ││
│  │  ✓ = clean  ⚠ = warnings   │  │    100MB threshold   ││
│  │  ✗ = violations             │  │                      ││
│  └─────────────────────────────┘  └─────────────────────┘│
│                                                          │
│  ┌── Summary ──────────────────────────────────────────┐│
│  │  Projects: 47  │  Clean: 28  │  Warnings: 12  │  Violations: 7  ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ↑↓ Navigate  │  → Expand  │  Space: Inspect  │  A: Audit All│
└──────────────────────────────────────────────────────────┘
```

**Components:** `ProjectTree` (with status icons), `ProjectDetailPanel`, `SummaryBar`

---

### 3.6 Hygiene — System Junk Management

```
┌──────────────────────────────────────────────────────────┐
│  HYGIENE                                Pending: 23 items │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌── Categories ────────────────┐  ┌── Items to Review ─┐│
│  │                               │  │                    ││
│  │  🐳 Docker           ██ 8    │  │  [ ] ubuntu:18.04   ││
│  │  💿 APFS Snapshots   █ 3     │  │   image, 3y old     ││
│  │  📦 Caches           ███ 10  │  │   127 MB            ││
│  │  🗑  Temp files       █ 2     │  │                     ││
│  │                               │  │  [ ] node:14-alpine ││
│  │  ───────────────────────      │  │   image, unused     ││
│  │  [Scan All] [Scan Selected]   │  │   112 MB            ││
│  │                               │  │                     ││
│  │                               │  │  [ ] exited-nginx   ││
│  │                               │  │   container         ││
│  │                               │  │   0 B (exited)      ││
│  │                               │  │                     ││
│  │                               │  │  ────────────────── ││
│  │                               │  │  Total selected:    ││
│  │                               │  │  8 items · 2.4 GB   ││
│  │                               │  │                     ││
│  │                               │  │  [Approve Selected] ││
│  │                               │  │  [Skip Selected]    ││
│  │                               │  │  [Approve All]      ││
│  └───────────────────────────────┘  └────────────────────┘│
│                                                          │
│  ┌── Item Detail ───────────────────────────────────────┐│
│  │  Category:  Docker Images                             ││
│  │  Path:      docker://ubuntu:18.04                     ││
│  │  Size:      127 MB                                    ││
│  │  Created:   2023-01-15                                ││
│  │  Status:    ⏳ Pending review                         ││
│  │                                                       ││
│  │  ⚠ This is an old base image. Verify it is not        ││
│  │    referenced by any active Dockerfile or compose.    ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ↑↓ Navigate  │  Space: Toggle  │  A: Approve  │  S: Skip │
│  C: Clean All Approved  │  Tab: Switch panels            │
└──────────────────────────────────────────────────────────┘
```

**Components:** `CategorySidebar`, `ItemChecklist`, `ItemDetailPanel`, `ApprovalActionBar`

### Hygiene Flow

1. User selects categories to scan → daemon populates `junk_items`
2. Items appear in "Items to Review" panel
3. User inspects each item (detail panel shows warnings, age, size)
4. User toggles individual items with `Space` or uses batch approve
5. User presses `C` to clean all approved items
6. Cleanup results logged to `cleanup_jobs`

---

### 3.7 Report — Statistics & History

```
┌──────────────────────────────────────────────────────────┐
│  REPORT                                                   │
│  ═══════════════════════════════════════════════════════ │
│                                                          │
│  ┌── Space Saved Over Time ────────────────────────────┐│
│  │                                                      ││
│  │  10G ┤                                        ╭────  ││
│  │   8G ┤                              ╭─────────╯      ││
│  │   6G ┤                    ╭─────────╯                ││
│  │   4G ┤          ╭─────────╯                          ││
│  │   2G ┤──────────╯                                    ││
│  │   0G ┼────┬────┬────┬────┬────┬────┬────             ││
│  │       Apr 07  14   21   28  May 05                   ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  ┌── Breakdown by Pattern ─────┐  ┌── By System ───────┐│
│  │  node_modules   ████████ 52% │  │  TM+CCC  ██████ 68%││
│  │  .venv          ████    24% │  │  TM only ██    12%││
│  │  .next          ██      12% │  │  CCC only███   20%││
│  │  build          █        7% │  │                     ││
│  │  other          █        5% │  │                     ││
│  └─────────────────────────────┘  └────────────────────┘│
│                                                          │
│  ┌── Daemon Uptime & Events ───────────────────────────┐│
│  │  Uptime: 14d 6h 32m  │  Events today: 47            ││
│  │  Projects detected: 3  │  Exclusions applied: 12     ││
│  │  Errors: 0             │  Last heartbeat: 2s ago     ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  1-6 Screens  │  E: Export  │  q Quit                    │
└──────────────────────────────────────────────────────────┘
```

**Components:** `SparklineChart`, `BarChart`, `PieChart` (or text-based equivalents), `StatsGrid`

---

## 4. Theme & Styling

### Color Palette (Lip Gloss adaptive)

```go
var Theme = struct {
    Primary   lipgloss.Color  // Cyan (#00BCD4 equivalent)
    Success   lipgloss.Color  // Green
    Warning   lipgloss.Color  // Yellow/Amber
    Danger    lipgloss.Color  // Red
    Muted     lipgloss.Color  // Gray
    Highlight lipgloss.Color  // Magenta/Pink
}{
    Primary:   lipgloss.Color("6"),   // Cyan
    Success:   lipgloss.Color("2"),   // Green
    Warning:   lipgloss.Color("3"),   // Yellow
    Danger:    lipgloss.Color("1"),   // Red
    Muted:     lipgloss.Color("8"),   // Bright Black (gray)
    Highlight: lipgloss.Color("5"),   // Magenta
}
```

### Visual hierarchy

- **Headers:** Bold, primary color background
- **Tables:** Alternating row colors, header row bold
- **Selected items:** Highlight color foreground, bold
- **Status indicators:** `●` green (running), `○` gray (stopped), `⚠` yellow, `✗` red
- **Borders:** Rounded, muted color
- **Focus:** Focused panel has brighter border

---

## 5. Responsive Layout

The TUI adapts to terminal size:

| Terminal Width | Layout |
|---|---|
| ≥ 120 cols | Dual-column panels |
| 80–119 cols | Single column, panels stack vertically |
| < 80 cols | Compact mode, truncated data, scrollable |

---

## 6. Keyboard Shortcuts Reference

| Key | Context | Action |
|---|---|---|
| `1`–`7` | Global | Switch to screen 1–7 |
| `Tab` | Global | Cycle focus between panels |
| `↑` `↓` / `j` `k` | Lists | Navigate items |
| `→` `←` / `l` `h` | Trees | Expand / collapse |
| `Space` | Lists, Trees | Toggle selection |
| `Enter` | Global | Confirm / Execute action |
| `?` | Global | Toggle help overlay |
| `q` / `Ctrl+C` | Global | Quit |
| `A` | Rules, Exclusions | Add new item |
| `E` | Rules, Exclusions | Edit selected |
| `D` | Rules, Exclusions | Delete selected |
| `R` | Exclusions | Restore selected exclusion |
| `F` | Exclusions, Audit | Open filter menu |
| `S` | Scanner | Start scan |

---

## 7. Component Library

All reusable TUI components are defined in `internal/tui/components/`:

| Component | File | Dependencies |
|---|---|---|
| `Tree` | `tree.go` | Custom recursive tree with icons, expand/collapse |
| `Table` | `table.go` | Wraps `bubbles/table` with sort, filter, selection |
| `Gauge` | `gauge.go` | Progress bar for scan operations |
| `Log` | `log.go` | Wraps `bubbles/viewport` with auto-scroll |
| `Spinner` | `spinner.go` | Wraps `bubbles/spinner` with custom styles |
| `StatusCard` | `card.go` | Bordered info block with icon + label + value |
| `FilterBar` | `filter.go` | Toggle-based filter row |
| `RuleEditor` | `editor.go` | Form with textinputs + dropdowns |

---

## 8. State Management (Bubble Tea Model)

```go
type MainModel struct {
    // Navigation
    currentScreen Screen
    screens       map[Screen]tea.Model

    // Shared state (read from DB via services)
    projects    []domain.Project
    exclusions  []domain.Exclusion
    rules       []domain.Rule
    events      []domain.DaemonEvent

    // Services (injected)
    projectService   *services.ProjectService
    exclusionService *services.ExcludeService
    ruleService      *services.RuleService

    // Dimensions
    width  int
    height int
    ready  bool
}
```

Each screen is a separate `tea.Model` that receives a subset of shared state. Screens communicate with the main model through messages.

---

## 9. No Mock Data

All data displayed in the TUI is sourced **exclusively** from the real SQLite database populated by the daemon and CLI operations. There are no fixtures, no seed data, no placeholder values. Empty states show "No data yet. Run `jart-stow scan` to start."
