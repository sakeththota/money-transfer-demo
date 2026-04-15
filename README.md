# Money Transfer Demo

A Temporal workflow demo showcasing money transfer scenarios, built with:
- **Backend**: Go (Temporal Worker + API Server)
- **Frontend**: Next.js with Tailwind CSS
- **Infrastructure**: Docker Compose

## Scenarios

The demo includes several execution scenarios:
* **Happy Path** - Everything works as intended
* **Advanced Visibility** - Updates a Search Attribute (Step) as it progresses
* **Human-In-Loop** - Requires signal approval with timeout
* **API Downtime** - Simulates unreliable API (recovers after 5th attempt)
* **Bug in Workflow** - Throws an error (fix and redeploy worker to recover)
* **Saga Rollback** - Demonstrates compensation on failure

## Quick Start

The easiest way to run the demo is with [just](https://github.com/casey/just):

```bash
just up
```

This starts all services in Docker containers:
- Temporal Server (http://localhost:8233)
- Go Worker
- Go API Server (http://localhost:7070)
- Next.js Frontend (http://localhost:3000)

Open http://localhost:3000 to use the demo.

## Using the Justfile

The project includes a `justfile` with commands for common tasks. Run `just` to see all available commands.

### Starting Services

```bash
just up                  # Start all services (local Temporal)
just up worker           # Start only the worker
just up-encrypted        # Start with payload encryption enabled
just down                # Stop all services
just restart api         # Restart a specific service
```

### Temporal Cloud

```bash
just cloud-setup         # Show setup instructions
just up-cloud            # Start connected to Temporal Cloud
just up-cloud-encrypted  # Start with encryption + Temporal Cloud
```

### Logs and Debugging

```bash
just logs                # Tail logs from all services
just logs worker         # Tail logs from a specific service
```

### Building and Cleanup

```bash
just build               # Build all containers
just rebuild worker      # Rebuild without cache
just clean               # Stop and remove containers
just nuke                # Full cleanup (containers, volumes, images)
```

## Manual Setup (Alternative)

If you prefer not to use `just`, you can run Docker Compose directly:

```bash
docker compose --profile local up -d
```

Or run services individually for development:

### Prerequisites
- Go 1.23+
- Node.js 20+
- Temporal CLI

### Start Services Manually

```bash
# Terminal 1: Temporal Server
temporal server start-dev --search-attribute Step=Keyword

# Terminal 2: Worker
cd go && ./startlocalworker.sh

# Terminal 3: API
cd go && ./startlocalapi.sh

# Terminal 4: Frontend
cd frontend && npm install && npm run dev
```

## Temporal Cloud

### Setup

1. Copy the example config:
   ```bash
   cp .env.cloud.example .env.cloud
   ```

2. Edit `.env.cloud` with your credentials:
   ```bash
   TEMPORAL_ADDRESS=your-namespace.your-account.tmprl.cloud:7233
   TEMPORAL_NAMESPACE=your-namespace.your-account
   
   # Option A: mTLS (place certs in ./certs/)
   TEMPORAL_CERT_PATH=/certs/client.pem
   TEMPORAL_KEY_PATH=/certs/client.key
   
   # Option B: API Key
   TEMPORAL_API_KEY=your-api-key
   ```

3. Start with:
   ```bash
   just up-cloud
   ```

## Payload Encryption

Enable end-to-end payload encryption:

```bash
just up-encrypted          # Local with encryption
just up-cloud-encrypted    # Cloud with encryption
```

When encryption is enabled, a codec server runs at http://localhost:8081. To decrypt payloads in the Temporal UI:
1. Open http://localhost:8233 (or https://cloud.temporal.io)
2. Click the settings gear icon
3. Set codec endpoint to `http://localhost:8081`

## Project Structure

```
money-transfer-demo/
├── go/
│   ├── activities/     # Temporal activities
│   ├── api/            # HTTP API server
│   ├── app/            # Shared types
│   ├── codec-server/   # Codec server for encryption
│   ├── encryption/     # Encryption utilities
│   ├── messages/       # Signals and queries
│   ├── worker/         # Temporal worker
│   └── workflows/      # Temporal workflows
├── frontend/
│   └── src/
│       ├── app/        # Next.js app router
│       ├── components/ # React components
│       └── lib/        # API client
├── justfile            # Task runner commands
├── compose.yaml        # Docker Compose config
└── .env.cloud.example  # Temporal Cloud config template
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/serverinfo` | GET | Get Temporal connection info |
| `/runWorkflow` | POST | Start a transfer workflow |
| `/runQuery` | POST | Query workflow status |
| `/approveTransfer` | POST | Send approval signal |
| `/listWorkflows` | GET | List recent workflows |
| `/listSchedules` | GET | List scheduled transfers |
| `/scheduleWorkflow` | POST | Create scheduled transfer |
| `/scheduleInfo/:id` | GET | Get schedule details |
| `/schedule/:id` | DELETE | Delete a schedule |
