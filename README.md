# Time MCP Server

> A Model Context Protocol server that provides comprehensive time-related capabilities including current time lookup, timezone conversion, daylight-saving detection, and natural-language date parsing. Available in **Go** and **Python** implementations with a full **TypeScript** client and test suite.

![Go](https://img.shields.io/badge/language-Go-blue)
![Python](https://img.shields.io/badge/language-Python-green)
![TypeScript](https://img.shields.io/badge/language-TypeScript-blue)
![License](https://img.shields.io/badge/license-MIT-blue)

## ✨ Features

* **Current Time Queries** – get the current time in any timezone  
* **Time-zone Conversions** – convert a HH:MM time between zones  
* **Daylight-Saving Detection** – know if a zone is in DST  
* **Time-difference Calculation** – hours offset when converting  
* **Natural-language Date Parsing** – "next Friday at noon", "3 days from now"  

## Available Tools

| Tool | Purpose | Arguments |
|------|---------|-----------|
| `get_current_time` | current time for a zone | `timezone` (string, optional) |
| `convert_time` | convert HH:MM between zones | `source_timezone` (string, required) • `time` (HH:MM, required) • `target_timezone` (string, required) |
| `parse_natural_time` | parse English date phrases | `expression` (string, required) • `timezone` (string, optional) |

## Project Structure
```

time-mcp-server/
├── main.go                    # Go server implementation
├── parse_natural_test.go      # Go tests with deterministic time injection
├── time_mcp_server.py         # Python server implementation  
├── run_time_server.sh         # Python server launcher script
└── ts/                        # TypeScript client & tests
    ├── src/mcp-client.ts      # Full-featured MCP client
    ├── tests/                 # Comprehensive test suite
    │   ├── parseNaturalTime.test.ts
    │   └── parseNaturalTimeEdgeCases.test.ts
    ├── package.json
    ├── jest.config.js
    └── tsconfig.json

FileEditView

Copy

Focus

```
## Quick Start

### 1. Go Server (Recommended)

```bash
# Build the server
go build -o time-mcp-server .

# Run with default settings (stdio transport)
./time-mcp-server

# Run with specific timezone and SSE transport
./time-mcp-server --local-timezone="America/New_York" --transport=sse --port=8080

# Run tests with deterministic time injection
go test -v
```

### 2. Python Server (Alternative)

FileEditView

Copy

Focus

```
# Install dependencies
pip install mcp fastmcp python-dateutil recurrent zoneinfo tzlocal

# Run directly
python time_mcp_server.py

# Or use the provided script
chmod +x run_time_server.sh
./run_time_server.sh
```

### 3. TypeScript Client & Tests

bash

FileEditView

Copy

Focus

```
cd ts/

# Install dependencies
npm install
# or
bun install

# Build the client
npm run build

# Run the client CLI
npm start

# Run comprehensive test suite
npm test

# Run tests in watch mode
npm run test:watch

# Run with coverage
npm run test:coverage
```

## Building & Cross-Compilation

### Go Server - Multi-platform Builds

bash

FileEditView

Copy

Focus

```
# Build for current platform
make

# Build for all platforms
make build-all

# Build for specific platforms
make build-darwin-universal    # macOS Universal Binary
make build-darwin-arm64        # macOS Apple Silicon
make build-linux-amd64         # Linux x86_64
make build-windows-amd64       # Windows x86_64

# Create distribution packages
make dist

# Clean build artifacts
make clean
```

Built binaries will be in 

```
.build/
```

 directory.

### TypeScript Client

bash

FileEditView

Copy

Focus

```
cd ts/

# Development build
npm run build

# Production build with optimizations
tsc --build --clean && tsc
```

## Testing

### Go Server Tests

The Go implementation includes sophisticated tests with deterministic time injection for reliable testing of time-dependent functionality:

bash

FileEditView

Copy

Focus

```
# Run all tests
go test -v

# Run specific test
go test -v -run TestTimeServerParseNatural_Deterministic

# Run with race detection
go test -race -v
```

Key test features:
- Fixed reference time injection for consistent results
- DST transition testing (spring forward/fall back)
- Timezone conversion validation
- Natural language parsing edge cases

### TypeScript Client Tests

Comprehensive Jest-based test suite covering happy paths and edge cases:

bash

FileEditView

Copy

Focus

```
cd ts/

# Run all tests
npm test

# Run specific test file
npm test parseNaturalTime.test.ts

# Run with verbose output
VERBOSE_TESTS=true npm test

# Watch mode for development
npm run test:watch

# Coverage report
npm run test:coverage
```

## Test features:

- Multi-path server discovery (handles different build locations)
- Process lifecycle management (proper cleanup)
- Error handling validation
- Timezone-aware assertions

## Command Line Options

### Go Server

bash

FileEditView

Copy

Focus

```
./time-mcp-server [OPTIONS]

Options:
-t, --transport string     Transport type: "stdio" or "sse" (default: "stdio")
-p, --port int            Port for SSE transport (default: 8080)
-l, --local-timezone string  Override detected local timezone
-v, --version             Show version and exit
-h, --help               Show help and exit
```

### TypeScript Client

bash

FileEditView

Copy

Focus

```
cd ts/

# Run client CLI
npm start

# Interactive mode (not yet implemented)
npm run interactive
```

## Integration Examples

### Claude Desktop

Add to your 

```
claude_desktop_config.json
```

:

json

FileEditView

Copy

Focus

```
{
"mcpServers": {
  "time-server": {
    "command": "/path/to/time-mcp-server",
    "args": ["--local-timezone", "America/Chicago"]
  }
}
}
```

### Cursor IDE

Add to your Cursor 

```
mcp.json
```

:

json

FileEditView

Copy

Focus

```
{
"mcpServers": {
  "time-server": {
    "command": "/path/to/time-mcp-server",
    "args": ["--transport", "stdio"]
  }
}
}
```

### Programmatic Usage (TypeScript)

typescript

FileEditView

Copy

Focus

```
import { MCPClient } from './ts/src/mcp-client';

const client = new MCPClient('/path/to/time-mcp-server', [
'--transport', 'stdio',
'--local-timezone', 'UTC'
]);

await client.connect();
await client.initialize();

// Get current time
const now = await client.getCurrentTime('America/New_York');

// Convert time between zones
const converted = await client.convertTime('UTC', '14:30', 'Asia/Tokyo');

// Parse natural language
const parsed = await client.parseNaturalTime('next Friday at 3pm', 'Europe/London');

client.disconnect();
```

## API Examples

### Current Time

Request:

json

FileEditView

Copy

Focus

```
{
"name": "get_current_time",
"arguments": { "timezone": "America/New_York" }
}
```

Response:

json

FileEditView

Copy

Focus

```
{
"timezone": "America/New_York",
"datetime": "2024-11-05T14:30:45-05:00",
"is_dst": false
}
```

### Time Conversion

Request:

json

FileEditView

Copy

Focus

```
{
"name": "convert_time",
"arguments": {
  "source_timezone": "America/Los_Angeles",
  "time": "15:30",
  "target_timezone": "Asia/Tokyo"
}
}
```

Response:

json

FileEditView

Copy

Focus

```
{
"source": {
  "timezone": "America/Los_Angeles",
  "datetime": "2024-11-05T15:30:00-08:00",
  "is_dst": false
},
"target": {
  "timezone": "Asia/Tokyo", 
  "datetime": "2024-11-06T08:30:00+09:00",
  "is_dst": false
},
"time_difference": "+17h"
}
```

### Natural Language Parsing

Request:

json

FileEditView

Copy

Focus

```
{
"name": "parse_natural_time",
"arguments": {
  "expression": "next Friday at noon",
  "timezone": "America/Chicago"
}
}
```

Response:

json

FileEditView

Copy

Focus

```
{
"timezone": "America/Chicago",
"datetime": "2024-11-08T12:00:00-06:00", 
"is_dst": false
}
```

## Development

### Prerequisites

- Go 1.24.3+ (for Go server)
- Python 3.8+ (for Python server)
- Node.js 18+ (for TypeScript client)
- Make (for build automation)

### Development Workflow

FileEditView

Copy

Focus

```
# 1. Clone and setup
git clone <repository>
cd time-mcp-server

# 2. Go development
go mod tidy
go build -o time-mcp-server .
go test -v

# 3. TypeScript development  
cd ts/
npm install
npm run build
npm test

# 4. Python development
pip install -r requirements.txt  # if you create one
python time_mcp_server.py
```

### Adding New Features

1. Go Server: Add to 

```
main.go
```

, update tests in 

```
parse_natural_test.go
```
2. TypeScript Client: Update 

```
src/mcp-client.ts
```

, add tests in 

```
tests/
```
3. Python Server: Modify 

```
time_mcp_server.py
```

## Troubleshooting

### Common Issues

Go build fails:

bash

FileEditView

Copy

Focus

```
go mod tidy
go clean -cache
```

TypeScript tests fail to find server:

bash

FileEditView

Copy

Focus

```
# Ensure server is built
go build -o time-mcp-server .
# Or check test paths in test files
```

Python server import errors:

bash

FileEditView

Copy

Focus

```
pip install --upgrade mcp fastmcp python-dateutil recurrent
```

### Debug Mode

Go server with verbose logging:

bash

FileEditView

Copy

Focus

```
./time-mcp-server --transport=sse --port=8080
# Check stderr for detailed logs
```

TypeScript tests with verbose output:

bash

FileEditView

Copy

Focus

```
VERBOSE_TESTS=true npm test
```

## License

MIT License - see LICENSE file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: 

```
go test -v && cd ts && npm test
```
5. Submit a pull request

Note: The Go implementation is the primary/recommended server due to its performance and comprehensive natural language parsing capabilities. The Python implementation provides an alternative for Python-centric environments.