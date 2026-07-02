# MiMo-Reasonix Desktop (Proof of Concept)

This is a minimal Wails proof of concept for the MiMo-Reasonix desktop application.

## Features

- Basic chat interface
- Submit messages to agent
- Cancel running turns
- View session history
- Real-time event streaming

## Prerequisites

- Go 1.21+
- Node.js 18+
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

## Development

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in development mode
wails dev

# Build for production
wails build
```

## Architecture

```
desktop/
├── main.go                 # Wails entry point + App bindings
├── wails.json              # Wails configuration
├── go.mod                  # Go module
├── frontend/
│   ├── src/
│   │   ├── App.tsx         # Main React app
│   │   ├── main.tsx        # Entry point
│   │   └── index.css       # Styles
│   ├── package.json
│   └── vite.config.ts
└── README.md
```

## Integration with MiMo-Reasonix Core

The desktop app communicates with the MiMo-Reasonix core via:

1. **Direct Go integration** - Import and use Controller directly
2. **HTTP/SSE serve mode** - Connect to running serve instance
3. **ACP (Agent Client Protocol)** - JSON-RPC over stdio

For this PoC, we simulate the agent responses. To integrate with the real core:

1. Import `mimo-reasonix/internal/control`
2. Create Controller instance in `main.go`
3. Bind Controller methods to App struct
4. Subscribe to event.Sink for real-time updates

## Next Steps

1. Integrate with real Controller
2. Add session management UI
3. Implement tool approval UI
4. Add settings panel
5. System tray integration
6. Auto-updater
