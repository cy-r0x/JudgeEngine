# judgenot0 Engine

The execution engine (judge daemon) for `judgenot0`. It processes code submissions, compiles, runs, and evaluates them securely using the `isolate` sandbox.

## Features
- Scalable, asynchronous job processing via RabbitMQ.
- Secure, resource-limited code execution.
- Multiple language support (C, C++, Python, Node.js).
- Strict space and flexible floating-point output comparison.

## Supported Operating Systems
- **Linux only**: The engine relies heavily on `isolate`, which requires Linux kernel features (namespaces, control groups (cgroups)) to sandbox execution successfully.

## Requirements

### Core Dependencies
- [Go](https://golang.org/dl/) (1.24+)
- **Message Broker**: [RabbitMQ](https://www.rabbitmq.com/) (`amqplib`)
- **Sandbox Environment**: [Isolate](https://github.com/ioi/isolate)

### Language Compilers & Runtimes
To evaluate code submissions, install the corresponding compilers and interpreters on the host system:
- **C/C++**: `gcc` and `g++` (supports `gnu11` and `gnu++23`)
- **Python**: `python3`
- **Node.js**: `node` or `nodejs`

## Installation

### 1. Install System Dependencies
Install compilers, runtimes, and essential tools required to compile both `isolate` and the submitted code:
```bash
sudo apt-get update
sudo apt-get install -y build-essential libcap-dev git gcc g++ python3 nodejs
```

### 2. Install Isolate
Clone the IOI isolate repository and install it. This provides the secure bounding boxes for code execution:
```bash
git clone https://github.com/ioi/isolate.git
cd isolate
make isolate
sudo make install
```

**Usermode & Cgroup v2 Configuration:**
To allow the judge daemon to run as a standard user (usermode) on modern Linux distributions using cgroup v2, you must link and enable the `isolate-cg-daemon` service. This daemon securely handles cgroup delegations without requiring the engine itself to run as root:

```bash
# Inside the isolate repository directory
sudo cp systemd/isolate-cgd.service /etc/systemd/system/

# Copy the systemd files to your system
sudo cp systemd/isolate.service /etc/systemd/system/
sudo cp systemd/isolate.scope /etc/systemd/system/

# Reload systemd to recognize the new files
sudo systemctl daemon-reload

# Enable and start the services
sudo systemctl enable --now isolate-cgd.service
sudo systemctl enable --now isolate.service
```

Ensure your system permits user namespaces (enabled by default on most modern distros):
```bash
sudo sysctl -w kernel.unprivileged_userns_clone=1
```

### 3. Setup RabbitMQ
If RabbitMQ is not already running, install it (e.g., via Docker):
```bash
docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
```

### 4. Build the Engine
Clone the repository and compile the Go daemon:
```bash
cd judgenot0/engine
go build ./...
```

## Configuration

Control the daemon's behavior by creating a `.env` file in the `engine` directory:

```env
# RabbitMQ Connection
RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
QUEUE_NAME="judge_queue"

# Judge Worker Settings
WORKER_COUNT=4            # Number of parallel isolate sandboxes
HTTP_PORT=8080            # Health/debug HTTP server port

# API Integration
ENGINE_KEY="your-engine-secret-key"
SERVER_ENDPOINT="http://localhost:3000/internal/verdict"  # Main server webhook endpoint
```

## Running the Engine
Start the daemon directly via Go, or execute the built binary:
```bash
go run .
# or
./engine
```

On startup, the daemon will:
1. Initialize the `isolate` sandboxes based on `WORKER_COUNT`.
2. Establish a connection to RabbitMQ.
3. Begin consuming and safely evaluating submissions from the queue.
