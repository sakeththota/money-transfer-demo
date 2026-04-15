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
* **Invalid Account** - Business failure (fails the workflow)
* **Saga Rollback** - Demonstrates compensation on failure

## Quick Start with Docker Compose

The easiest way to run the demo:

```bash
docker compose up
```

This starts:
- Temporal Server (port 8233 for UI)
- Go Worker
- Go API Server (port 7070)
- Next.js Frontend (port 3000)

Open http://localhost:3000 to use the demo.

## Running Locally (Development)

### Prerequisites
- Go 1.23+
- Node.js 20+
- Temporal CLI

### 1. Start Temporal Server

```bash
temporal server start-dev --search-attribute Step=Keyword
```

### 2. Start the Go Worker

```bash
cd go
./startlocalworker.sh
```

### 3. Start the Go API Server

```bash
cd go
./startlocalapi.sh
```

### 4. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:3000

## Running on Temporal Cloud

### 1. Configure Cloud Environment

Copy and edit the environment file:

```bash
cp setcloudenv.example setcloudenv.sh
```

Edit `setcloudenv.sh`:

```bash
# Using mTLS
export TEMPORAL_ADDRESS=<namespace>.<accountID>.tmprl.cloud:7233
export TEMPORAL_NAMESPACE=<namespace>.<accountID>
export TEMPORAL_CERT_PATH="/path/to/cert.pem"
export TEMPORAL_KEY_PATH="/path/to/key.key"

# Or using API keys
export TEMPORAL_ADDRESS=<region>.<cloud_provider>.api.temporal.io:7233
export TEMPORAL_NAMESPACE=<namespace>.<accountID>
export TEMPORAL_API_KEY=<api_key>
```

### 2. Create Search Attribute

```bash
tcld login
tcld namespace search-attributes add --namespace <namespace>.<accountId> --search-attribute "Step=Keyword"
```

### 3. Start Services

```bash
# Terminal 1: Worker
cd go && ./startcloudworker.sh

# Terminal 2: API
cd go && ./startcloudapi.sh

# Terminal 3: Frontend
cd frontend && npm run dev
```

## Using Encryption

Enable payload encryption by setting `ENCRYPT_PAYLOADS=true`:

```bash
# Worker
cd go && ./startlocalworker.sh true

# API
cd go && ./startlocalapi.sh true
```

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
└── compose.yaml        # Docker Compose config
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/serverinfo` | GET | Get Temporal connection info |
| `/runWorkflow` | POST | Start a transfer workflow |
| `/runQuery` | POST | Query workflow status |
| `/approveTransfer` | POST | Send approval signal |
| `/listWorkflows` | GET | List recent workflows |
| `/scheduleWorkflow` | POST | Create scheduled transfer |
