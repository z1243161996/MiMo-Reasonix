# Desktop Framework Evaluation: Wails vs Tauri

> Date: 2026-07-01
> Author: MiMo-Reasonix Team
> Status: Draft

## Executive Summary

This document evaluates Wails (Go) and Tauri (Rust) as desktop framework options for MiMo-Reasonix. Based on the existing Go codebase, architecture patterns, and practical considerations, **Wails is the recommended choice** for the desktop implementation.

## 1. Current Architecture Analysis

### 1.1 MiMo-Reasonix Core

MiMo-Reasonix is a Go-based coding agent with:

- **51 internal packages** under `internal/`
- **Controller pattern** (`internal/control/controller.go`) abstracting frontends
- **Event system** (`internal/event/`, `internal/eventwire/`) for typed event streaming
- **HTTP/SSE serve mode** (`internal/serve/`) with 27+ REST endpoints
- **ACP (Agent Client Protocol)** over JSON-RPC 2.0 for editor integration
- **Single static binary** with `CGO_ENABLED=0`

### 1.2 Existing Frontend Abstraction

The Controller pattern already supports multiple frontends:

```go
// Package control is the transport-agnostic session driver.
// A Controller owns the agent run loop and session lifecycle,
// takes commands (Send/Cancel/Approve/SetPlanMode/Compact/NewSession/…),
// and emits everything that happens as a typed event stream to a single event.Sink.
```

**Supported frontends:**
1. **CLI TUI** - Charm bubbletea/lipgloss (current)
2. **HTTP/SSE serve** - REST API + SSE streaming (current)
3. **Wails desktop** - Referenced in docs, planned (future)

### 1.3 HTTP/SSE Serve Mode

The serve mode provides a complete HTTP API:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/events` | GET | SSE event stream |
| `/submit` | POST | Submit user input |
| `/cancel` | POST | Cancel running turn |
| `/approve` | POST | Approve tool calls |
| `/history` | GET | Session message history |
| `/context` | GET | Context window usage |
| `/status` | GET | Session status |
| `/sessions` | GET | List all sessions |
| `/skills` | GET | List available skills |
| `/todos` | GET | List todo items |
| `/checkpoints` | GET | List checkpoints |
| `/branches` | GET | List branches |
| `/plan` | POST | Toggle plan mode |
| `/compact` | POST | Compact context |
| `/new` | POST | New session |
| `/rewind` | POST | Rewind to checkpoint |
| `/fork` | POST | Fork session |
| `/summarize` | POST | Summarize turn |
| `/resume` | POST | Resume session |
| `/forget` | POST | Delete memory |
| `/goal` | POST | Set/clear goal |
| `/answer` | POST | Answer ask request |
| `/tool-approval-mode` | POST | Set approval mode |
| `/auto-approve-tools` | POST | Toggle YOLO mode |
| `/bypass` | POST | Legacy YOLO endpoint |
| `/delete-session` | POST | Delete session |

### 1.4 Event System

Typed events for UI rendering:

```go
type Kind int

const (
    TurnStarted Kind = iota
    Reasoning          // Thinking-mode reasoning delta
    Text               // Answer-text delta
    Message            // Complete assistant message
    ToolDispatch       // Tool call about to run
    ToolResult         // Finished tool call
    Usage              // Token telemetry
    Notice             // Out-of-band message
    Phase              // Coordinator boundary
    ApprovalRequest    // Approve pending tool call
    AskRequest         // Structured questions
    TurnDone           // End of turn
    CompactionStarted  // Context compaction start
    CompactionDone     // Context compaction done
    ToolProgress       // Live tool output
    MCPSurfaceReady    // MCP surface loaded
    Retrying           // Provider retry
    Steer              // Mid-turn steer
    // ... more
)
```

JSON wire format via `eventwire` package:

```go
type Event struct {
    Kind            string           `json:"kind"`
    Text            string           `json:"text,omitempty"`
    Reasoning       string           `json:"reasoning,omitempty"`
    Tool            *Tool            `json:"tool,omitempty"`
    Usage           *Usage           `json:"usage,omitempty"`
    Approval        *Approval        `json:"approval,omitempty"`
    Ask             *Ask             `json:"ask,omitempty"`
    Compaction      *Compaction      `json:"compaction,omitempty"`
    // ...
}
```

## 2. Framework Comparison

### 2.1 Wails (Go)

**Overview:** Wails is a Go framework for building desktop applications with web frontends. It uses webview for rendering and provides Go-to-JavaScript bindings.

**Architecture:**
```
┌─────────────────────────────────────────┐
│              Wails App                   │
├─────────────────────────────────────────┤
│  Frontend (HTML/CSS/JS/React/Vue)       │
│  ┌─────────────────────────────────┐   │
│  │  Webview (OS native)            │   │
│  └─────────────────────────────────┘   │
├─────────────────────────────────────────┤
│  Backend (Go)                           │
│  ┌─────────────────────────────────┐   │
│  │  Your Go code                   │   │
│  │  - Controller integration       │   │
│  │  - Event system                 │   │
│  │  - Business logic               │   │
│  └─────────────────────────────────┘   │
├─────────────────────────────────────────┤
│  Wails Runtime (Go ↔ JS bindings)      │
└─────────────────────────────────────────┘
```

**Key Features:**
- Native Go backend (no Rust required)
- Direct function call bindings (Go ↔ JS)
- Event emission from Go to JS
- Lifecycle management
- Built-in dev server with hot reload
- Cross-platform (macOS, Windows, Linux)
- Small binary size (~10-15MB)

**Advantages for MiMo-Reasonix:**
1. **Native Go integration** - No FFI, no separate language
2. **Direct Controller access** - Call Controller methods directly from Go
3. **Event system reuse** - Existing `event.Sink` can emit to webview
4. **Serve mode compatibility** - Can reuse HTTP/SSE serve mode logic
5. **Single codebase** - Go backend + web frontend
6. **Fast compilation** - Go compilation is faster than Rust
7. **Existing references** - Project already mentions Wails in docs

**Disadvantages:**
1. **Webview dependency** - Uses OS webview (may have inconsistencies)
2. **Smaller ecosystem** - Less mature than Tauri
3. **Limited native UI** - Web-based UI only
4. **CGO required** - Some features need CGO (but can be avoided)

### 2.2 Tauri (Rust)

**Overview:** Tauri is a Rust framework for building desktop applications with web frontends. It uses webview2 (Windows) or WKWebView (macOS) for rendering.

**Architecture:**
```
┌─────────────────────────────────────────┐
│              Tauri App                   │
├─────────────────────────────────────────┤
│  Frontend (HTML/CSS/JS/React/Vue)       │
│  ┌─────────────────────────────────┐   │
│  │  Webview2/WKWebView             │   │
│  └─────────────────────────────────┘   │
├─────────────────────────────────────────┤
│  Backend (Rust)                         │
│  ┌─────────────────────────────────┐   │
│  │  Rust code                      │   │
│  │  - IPC handlers                 │   │
│  │  - System API                   │   │
│  │  - Plugin system                │   │
│  └─────────────────────────────────┘   │
├─────────────────────────────────────────┤
│  Tauri IPC (Rust ↔ JS via JSON-RPC)    │
└─────────────────────────────────────────┘
```

**Key Features:**
- Rust backend (memory safety, performance)
- JSON-RPC based IPC
- Plugin system
- System tray, menu bar, window management
- Auto-updater
- Cross-platform (macOS, Windows, Linux)
- Small binary size (~5-10MB)

**Advantages for MiMo-Reasonix:**
1. **Memory safety** - Rust guarantees
2. **Performance** - Near-native performance
3. **Rich ecosystem** - More plugins and tools
4. **Better webview** - webview2 on Windows is more modern
5. **Security** - Rust's safety guarantees
6. **Active development** - Larger community

**Disadvantages for MiMo-Reasonix:**
1. **Rust required** - Need to rewrite Go backend in Rust
2. **FFI complexity** - Go ↔ Rust interop is complex
3. **Build time** - Rust compilation is slower
4. **Binary size** - Larger than Go binaries
5. **Two languages** - Go core + Rust desktop = maintenance overhead
6. **No direct Controller access** - Must bridge Go ↔ Rust ↔ JS

## 3. Integration Patterns

### 3.1 Wails Integration

**Direct Go integration:**

```go
// desktop/main.go
package main

import (
    "mimo-reasonix/internal/control"
    "mimo-reasonix/internal/event"
    "github.com/wailsapp/wails/v2"
)

type App struct {
    ctrl *control.Controller
    bc   *serve.Broadcaster
}

// Bind Controller methods directly
func (a *App) Submit(input string) {
    a.ctrl.SubmitHTTP(input)
}

func (a *App) Cancel() {
    a.ctrl.Cancel()
}

func (a *App) Approve(id string, allow bool) {
    a.ctrl.Approve(id, allow, false, false)
}

// Event emission
func (a *App) OnStartup(ctx context.Context) {
    // Subscribe to event system
    ch, _ := a.bc.Subscribe()
    go func() {
        for data := range ch {
            wailsruntime.EventsEmit(ctx, "agent:event", string(data))
        }
    }()
}

func main() {
    app := wails.Run(&app.Options{
        Title:  "MiMo-Reasonix",
        Width:  1200,
        Height: 800,
        OnStartup: app.OnStartup,
        Bind: []interface{}{
            app, // Expose all App methods to JS
        },
    })
}
```

**Frontend usage:**

```typescript
// React/Vue component
import { Submit, Cancel, Approve } from '../wailsjs/go/main/App';

function ChatInput() {
    const handleSubmit = async (input: string) => {
        await Submit(input);
    };

    const handleCancel = async () => {
        await Cancel();
    };

    return (
        <div>
            <input onKeyDown={(e) => {
                if (e.key === 'Enter') handleSubmit(e.target.value);
            }} />
            <button onClick={handleCancel}>Cancel</button>
        </div>
    );
}
```

### 3.2 Tauri Integration

**Rust backend with Go FFI:**

```rust
// src-tauri/src/main.rs
use tauri::command;

// Must bridge to Go via FFI
#[tauri::command]
fn submit(input: String) -> Result<(), String> {
    // Call Go via FFI (complex)
    unsafe { go_submit(input.as_ptr()) };
    Ok(())
}

#[tauri::command]
fn cancel() -> Result<(), String> {
    unsafe { go_cancel() };
    Ok(())
}
```

**Go FFI layer:**

```go
// desktop/ffi.go
package main

import "C"

//export go_submit
func go_submit(input *C.char) {
    ctrl.SubmitHTTP(C.GoString(input))
}

//export go_cancel
func go_cancel() {
    ctrl.Cancel()
}
```

**Frontend usage:**

```typescript
import { invoke } from '@tauri-apps/api/tauri';

function ChatInput() {
    const handleSubmit = async (input: string) => {
        await invoke('submit', { input });
    };

    const handleCancel = async () => {
        await invoke('cancel');
    };
    // ...
}
```

## 4. Performance Comparison

| Metric | Wails | Tauri |
|--------|-------|-------|
| **Binary Size** | ~10-15MB | ~5-10MB |
| **Startup Time** | ~100-200ms | ~50-100ms |
| **Memory Usage** | ~50-100MB | ~30-60MB |
| **IPC Latency** | ~1-5ms | ~0.5-2ms |
| **Build Time** | ~10-30s | ~30-120s |
| **Compilation** | Go (fast) | Rust (slow) |

**Analysis:**
- Tauri has better raw performance metrics
- Wails has faster development cycle (Go compilation)
- For a coding agent, IPC latency difference is negligible
- Memory usage difference is minimal for desktop apps

## 5. Development Effort Estimation

### 5.1 Wails Integration

| Task | Effort | Notes |
|------|--------|-------|
| Project setup | 1-2 days | Wails init, frontend scaffold |
| Controller binding | 2-3 days | Map Controller methods to JS |
| Event streaming | 2-3 days | SSE → Wails events |
| UI implementation | 5-10 days | Chat, settings, sessions |
| Testing | 3-5 days | Unit + integration |
| **Total** | **13-23 days** | |

**Key advantages:**
- Direct Go integration (no FFI)
- Reuse existing serve mode logic
- Event system directly accessible
- Fast iteration with hot reload

### 5.2 Tauri Integration

| Task | Effort | Notes |
|------|--------|-------|
| Project setup | 1-2 days | Tauri init, frontend scaffold |
| Rust backend | 5-10 days | IPC handlers, system API |
| Go FFI layer | 5-8 days | Bridge Go ↔ Rust |
| Event streaming | 3-5 days | Go → Rust → JS pipeline |
| UI implementation | 5-10 days | Chat, settings, sessions |
| Testing | 5-8 days | Unit + integration + FFI |
| **Total** | **24-43 days** | |

**Key disadvantages:**
- Two languages (Go + Rust)
- Complex FFI bridging
- Longer build times
- More testing surface

## 6. Risk Assessment

### 6.1 Wails Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Webview inconsistencies | Medium | Test on all platforms |
| Smaller community | Low | Core features stable |
| CGO dependency | Low | Use `CGO_ENABLED=0` where possible |
| Performance overhead | Low | Acceptable for desktop app |

### 6.2 Tauri Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Rust learning curve | High | Team needs Rust expertise |
| FFI complexity | High | May introduce bugs |
| Build time overhead | Medium | Affects development speed |
| Maintenance burden | High | Two codebases to maintain |
| Go core incompatibility | High | Must maintain Go ↔ Rust bridge |

## 7. Recommendation

### 7.1 Primary Recommendation: Wails

**理由 (Rationale):**

1. **Native Go integration** - MiMo-Reasonix is a Go project. Wails allows direct integration without FFI complexity.

2. **Existing architecture alignment** - The Controller pattern and event system are designed for frontend abstraction. Wails fits naturally.

3. **Serve mode reuse** - The HTTP/SSE serve mode can be directly used or adapted for Wails.

4. **Development speed** - Go compilation is faster than Rust, enabling quicker iteration.

5. **Lower risk** - No FFI bridging means fewer potential bugs.

6. **Existing references** - The project already mentions Wails in documentation (MIMO-REASONIX.md, docs/CHECKPOINTS.md, docs/GUIDE.md).

### 7.2 Implementation Strategy

**Phase 1: Foundation (Week 1-2)**
- Set up Wails project structure
- Bind Controller methods to JS
- Implement basic event streaming
- Create minimal chat UI

**Phase 2: Core Features (Week 3-4)**
- Session management (list, resume, delete)
- Tool approval UI
- Settings panel
- Model switching

**Phase 3: Polish (Week 5-6)**
- System tray integration
- Auto-updater
- Keyboard shortcuts
- Theme support

**Phase 4: Advanced Features (Week 7-8)**
- MCP management UI
- Memory panel
- Skills browser
- Checkpoint/branch visualization

### 7.3 Alternative: Tauri (If Rust Expertise Available)

If the team has strong Rust expertise and prefers Tauri's performance characteristics:

1. Keep Go core as a subprocess
2. Use Tauri's sidecar feature to run the Go binary
3. Communicate via HTTP/localhost (reuse serve mode)
4. Avoid direct FFI complexity

```
┌─────────────────────────────────────────┐
│              Tauri App                   │
├─────────────────────────────────────────┤
│  Frontend (React/Vue)                   │
├─────────────────────────────────────────┤
│  Rust Backend                           │
│  - Window management                    │
│  - System tray                          │
│  - Auto-updater                         │
├─────────────────────────────────────────┤
│  Sidecar: mimo-reasonix serve           │
│  - Go binary running HTTP/SSE           │
│  - localhost:PORT communication         │
└─────────────────────────────────────────┘
```

This approach:
- Avoids FFI complexity
- Reuses existing serve mode
- Keeps Go core intact
- Adds Tauri's UI capabilities

## 8. Conclusion

**Wails is the recommended choice** for MiMo-Reasonix desktop implementation due to:

1. Native Go integration (no FFI)
2. Alignment with existing architecture
3. Faster development cycle
4. Lower risk and maintenance burden
5. Existing project references

The estimated development effort is **13-23 days** for a functional desktop app, compared to **24-43 days** for Tauri with Go FFI.

For teams with strong Rust expertise, Tauri with sidecar architecture is a viable alternative that preserves the Go core while adding Tauri's UI capabilities.

---

## Appendix A: Wails Project Structure

```
desktop/
├── main.go                 # Wails entry point
├── app.go                  # App struct with Controller binding
├── events.go               # Event streaming logic
├── wails.json              # Wails configuration
├── frontend/
│   ├── src/
│   │   ├── App.tsx         # Main React/Vue app
│   │   ├── components/
│   │   │   ├── Chat.tsx    # Chat interface
│   │   │   ├── Sidebar.tsx # Session list
│   │   │   ├── Settings.tsx# Settings panel
│   │   │   └── ...
│   │   └── wailsjs/
│   │       └── go/
│   │           └── main/
│   │               └── App.ts # Generated bindings
│   ├── package.json
│   └── vite.config.ts
└── go.mod
```

## Appendix B: Tauri Sidecar Structure

```
desktop/
├── src-tauri/
│   ├── src/
│   │   └── main.rs         # Tauri entry point
│   ├── tauri.conf.json     # Tauri configuration
│   └── Cargo.toml
├── sidecar/
│   └── mimo-reasonix       # Go binary (built separately)
├── frontend/
│   ├── src/
│   │   ├── App.tsx
│   │   └── ...
│   ├── package.json
│   └── vite.config.ts
└── README.md
```

## Appendix C: References

- [Wails Documentation](https://wails.io/docs)
- [Tauri Documentation](https://tauri.app/v1/docs)
- [MiMo-Reasonix Architecture](./SPEC.md)
- [Serve Mode API](../internal/serve/serve.go)
- [Event System](../internal/event/event.go)
- [Controller Pattern](../internal/control/controller.go)
