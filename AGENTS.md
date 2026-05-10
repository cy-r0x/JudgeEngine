# judgenot0 Engine — Agent Guide

## Project Overview

This is the **judgenot0 Engine** (a.k.a. `judge-deamon`), the execution daemon for the `judgenot0` online judge system. It is a Go application that consumes code-submission jobs from a RabbitMQ queue, compiles and runs them inside the [`isolate`](https://github.com/ioi/isolate) sandbox, compares output against expected results, and reports verdicts back to a main server via an authenticated HTTP PATCH request.

The daemon also exposes a small HTTP server for health/metrics (Prometheus) and a manual `POST /run` endpoint.

## Technology Stack

- **Language**: Go 1.24.3+
- **Message Broker**: RabbitMQ (via `github.com/rabbitmq/amqp091-go`)
- **Sandbox**: [IOI Isolate](https://github.com/ioi/isolate) (Linux-only, requires cgroups v2 and user namespaces)
- **Metrics**: Prometheus client + `gopsutil/v4` for system-level gauges/counters
- **Configuration**: Environment variables loaded from `.env` via `github.com/joho/godotenv`

### Supported Languages

| Language | Runner | Compiler / Interpreter |
|----------|--------|------------------------|
| C        | `languages.C`    | `gcc --std=gnu11` |
| C++      | `languages.CPP`  | `g++ --std=gnu++23` |
| Python   | `languages.Python` | `/usr/bin/python3` |
| Node.js  | `languages.NodeJS` | `/usr/bin/node` or `/usr/bin/nodejs` |

## Project Structure

```
.
├── main.go              # Entry point: wires config, queue, scheduler, HTTP server, graceful shutdown
├── go.mod / go.sum      # Go module files (module: github.com/judgenot0/judge-deamon)
├── config/
│   └── config.go        # Singleton Config struct loaded from .env
├── cmd/
│   ├── serve.go         # HTTP server setup (Listen, Shutdown, logging middleware)
│   ├── routes.go        # Route registration (POST /run, GET /metrics)
│   ├── run.go           # Handler for POST /run (manual submission execution)
│   ├── register_node.go # Self-registration with the main server (Prometheus target discovery)
│   └── prom.go          # Prometheus system metrics collection (CPU, mem, disk, net)
├── handlers/
│   ├── handler.go       # Handler struct (holds Config)
│   ├── verdict.go       # ProduceVerdict: builds HMAC-signed payload and PATCHes the server
│   ├── compare.go       # Strict / whitespace-ignoring text comparison via `diff`
│   ├── compare_float.go # Token-wise float comparison with configurable epsilon
│   ├── parse_meta.go    # Parses isolate `meta.txt` to determine TLE/MLE/RE/IE
│   └── types.go         # Meta struct (isolate metadata fields)
├── languages/
│   ├── c.go             # C runner: Compile + Run inside isolate
│   ├── cpp.go           # C++ runner
│   ├── python.go        # Python runner
│   └── nodejs.go        # Node.js runner
├── queue/
│   ├── amqp.go          # Queue struct, InitQueue, Close
│   ├── connection.go    # RabbitMQ connect/reconnect with DLX/DLQ setup and exponential backoff
│   └── consumer.go      # StartConsume: message loop, unmarshals Submission, dispatches to scheduler
├── scheduler/
│   └── scheduler.go     # Worker pool, isolate sandbox init/cleanup, job orchestration
├── structs/
│   ├── submission.go    # Submission + Testcase structs (JSON tags)
│   ├── verdict.go       # Verdict struct
│   └── worker.go        # Worker struct (Id int)
└── utils/
    └── sendResponse.go  # JSON HTTP response helper
```

## Build & Run

### Prerequisites

- Linux host (Isolate requires Linux namespaces/cgroups)
- Go 1.24+
- RabbitMQ running and accessible
- Isolate installed and the `isolate-cg-daemon` systemd service enabled (for usermode / cgroup v2)
- Compilers/runtimes on the host: `gcc`, `g++`, `python3`, `node` (or `nodejs`)

### Build

```bash
go build ./...
```

### Run

```bash
cp .env.example .env
# edit .env with your values
go run .
# or after building:
./engine
```

### Configuration (`.env`)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `RABBITMQ_URL` | No | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection string |
| `QUEUE_NAME` | No | `judge_queue` | Queue to consume |
| `WORKER_COUNT` | No | `1` | Number of parallel isolate sandboxes |
| `HTTP_PORT` | No | `8080` | HTTP server port |
| `ENGINE_KEY` | **Yes** | — | Shared secret for HMAC signing verdicts |
| `SERVER_ENDPOINT` | **Yes** | — | Main server base URL (e.g. `http://localhost:8000`) |

## Runtime Architecture

1. **Startup**
   - Load `.env` into a singleton `Config`.
   - Initialize RabbitMQ connection, channel, queue, DLX, DLQ, and QoS.
   - Initialize `WORKER_COUNT` isolate sandboxes (`isolate --cg --init`).
   - Start the HTTP server (`:8080` by default) and register Prometheus metrics.
   - Register this node with the main server (`/register_node`) for scraping.

2. **Job Consumption**
   - The queue consumer loops forever, reconnecting on channel/connection loss.
   - Each RabbitMQ message is unmarshaled into a `structs.Submission`.
   - A worker (sandbox) is pulled from the `WorkChannel` pool.
   - `scheduler.Work` runs the submission in a goroutine.

3. **Execution Flow**
   - `GetRunner(language)` returns the language-specific `Runner`.
   - `Compile(ctx, boxId, submission)` writes source to `/var/local/lib/isolate/{boxId}/box/` and compiles.
   - `Run(ctx, boxId, submission, handler)` iterates testcases, writes `in.txt`/`expOut.txt`, invokes `isolate --run`, then calls the appropriate comparator.
   - After execution the sandbox is `--cleanup` + `--init` before the worker is returned to the pool.

4. **Verdict Reporting**
   - `ProduceVerdict` constructs an `EnginePayload` with HMAC-SHA256 token (`ENGINE_KEY`).
   - Sends an authenticated `PATCH` to `{SERVER_ENDPOINT}/api/submissions`.
   - If the PATCH fails, the RabbitMQ message is **Nack**ed (sent to DLQ); otherwise it is **Ack**ed.

5. **Graceful Shutdown**
   - On `SIGINT`/`SIGTERM`, cancel the consumer context, shutdown the HTTP server (30s timeout), close the queue connection, and exit.

## Output Comparison Strategies

- **Default (`checkerType != "float"`)**: Uses `diff` (strict) or `diff -Z -B` (ignore trailing/leading whitespace) depending on `CheckerStrictSpace`.
- **Float (`checkerType == "float"`)**: Tokenizes line-by-line, compares numeric tokens with tolerance `epsilon * (1 + max(|a|,|b|))`. Non-numeric tokens are compared exactly. Default epsilon is `1e-6` if `CheckerPrecision` is unset or invalid.

## Verdict Mapping

Internal codes → human-readable names sent to the server:

| Code | Meaning |
|------|---------|
| `ac`  | ACCEPTED |
| `wa`  | WRONG_ANSWER |
| `ce`  | COMPILATION_ERROR |
| `tle` | TIME_LIMIT_EXCEEDED |
| `mle` | MEMORY_LIMIT_EXCEEDED |
| `re`  | RUNTIME_ERROR |
| `ie`  | Internal Error (used for unexpected failures) |

## HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/run` | Manually execute a single submission (JSON body matching `Submission`) |
| `GET`  | `/metrics` | Prometheus metrics endpoint |

## Development Conventions

- **Logging**: Use `log.Printf` for all operational logging. The HTTP middleware logs method, URI, and duration.
- **Error Handling**: Language runners log errors and return appropriate verdicts (`ce`, `ie`) rather than crashing the daemon. Panics in worker goroutines are recovered and result in a Nack.
- **File Permissions**: Source and data files inside sandboxes are written with `0644`.
- **Sandbox Paths**: All language runners hard-code `/var/local/lib/isolate/{boxId}/box/` as the working directory.
- **Isolate Flags**: Always use `--cg` (cgroups mode). Wall-time is set to `1.5×` the problem time limit. File size limit is `10240` KB. Memory limit is passed as `--cg-mem` in KB (`MemoryLimit × 1024`).
- **No Tests**: The repository currently contains no unit tests or integration tests.

## Security Considerations

- **Sandboxing**: Execution happens exclusively inside `isolate` sandboxes. The engine itself does not run submitted code directly.
- **Authentication**: Verdict payloads are signed with HMAC-SHA256 using `ENGINE_KEY`. The server must validate the `Authorization: Bearer <token>` header.
- **CORS**: The `/run` endpoint sets `Access-Control-Allow-Origin: *`.
- **Input Validation**: Minimal — empty `language` or `sourceCode` results in `ce`. Malformed queue messages are Nack'd to the DLQ.
- **Secrets**: `ENGINE_KEY` and `RABBITMQ_URL` contain credentials and must be kept out of version control (`.gitignore` already ignores `.env`).

## Common Tasks

### Adding a New Language

1. Create `languages/<lang>.go` implementing the `Runner` interface:
   ```go
   type Runner interface {
       Compile(ctx context.Context, boxId int, runReq *structs.Submission) (structs.Verdict, error)
       Run(ctx context.Context, boxId int, runReq *structs.Submission, handler *handlers.Handler) structs.Verdict
   }
   ```
2. Register it in `scheduler/scheduler.go` inside `GetRunner`.
3. Ensure the compiler/interpreter is installed on the host system.

### Modifying Isolate Flags

Edit the `exec.CommandContext` calls in the respective `languages/*.go` `Run` methods. Common flags:
- `--time` (CPU time limit)
- `--wall-time` (wall-clock limit)
- `--cg-mem` (memory limit)
- `--fsize` (output file size limit)
- `--processes` (process limit — already set to `16` for Node.js)

### Adjusting Metrics Collection

System metrics are collected in `cmd/prom.go` (`SystemMetrics.Collect`) every 5 seconds. Add new Prometheus collectors there and register them in `Server.RegisterMetrics`.
