# Money Transfer Demo - Justfile
# Run `just` to see all available commands

set quiet

[private]
default:
    @just --list

# ═══════════════════════════════════════════════════════════════════════════════
# QUICK START
# ═══════════════════════════════════════════════════════════════════════════════

# Start: `just up` or `just up <worker|api|frontend|temporal|codec-server>`
[group('start')]
up *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        echo "Starting Temporal server..."
        docker compose --profile local up -d temporal
        echo "Waiting for Temporal to be healthy..."
        until docker compose --profile local exec -T temporal temporal operator cluster health 2>/dev/null | grep -q SERVING; do
            sleep 2
        done
        echo "Temporal is ready. Starting other services..."
        docker compose --profile local up -d
        echo ""
        echo "Services started (LOCAL mode)"
        echo "  Frontend:    http://localhost:3000"
        echo "  Temporal UI: http://localhost:8233"
        echo "  API:         http://localhost:7070"
        echo ""
    elif [ "{{service}}" = "temporal" ]; then
        docker compose --profile local up -d temporal
        echo "Waiting for Temporal to be healthy..."
        until docker compose --profile local exec -T temporal temporal operator cluster health 2>/dev/null | grep -q SERVING; do
            sleep 2
        done
        echo "Temporal is ready: http://localhost:8233"
    else
        docker compose --profile local up -d {{service}}
    fi

# Start with payload encryption enabled
[group('start')]
up-encrypted *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        echo "Starting Temporal server..."
        docker compose --profile local up -d temporal
        echo "Waiting for Temporal to be healthy..."
        until docker compose --profile local exec -T temporal temporal operator cluster health 2>/dev/null | grep -q SERVING; do
            sleep 2
        done
        echo "Temporal is ready. Starting other services with encryption..."
        ENCRYPT_PAYLOADS=true docker compose --profile local up -d
        echo ""
        echo "Services started with ENCRYPTION ENABLED (LOCAL mode)"
        echo "  Frontend:     http://localhost:3000"
        echo "  Temporal UI:  http://localhost:8233"
        echo "  API:          http://localhost:7070"
        echo "  Codec Server: http://localhost:8081"
        echo ""
        echo "To decrypt payloads in Temporal UI:"
        echo "  1. Open http://localhost:8233"
        echo "  2. Click settings (gear icon)"
        echo "  3. Set codec endpoint: http://localhost:8081"
        echo ""
    else
        ENCRYPT_PAYLOADS=true docker compose --profile local up -d {{service}}
    fi

# Stop: `just down` or `just down <service>`
[group('start')]
down *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        docker compose --profile local down
    else
        docker compose --profile local stop {{service}}
    fi

# Restart: `just restart` or `just restart <service>`
[group('start')]
restart *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        docker compose restart
    else
        docker compose restart {{service}}
    fi

# ═══════════════════════════════════════════════════════════════════════════════
# TEMPORAL CLOUD
# ═══════════════════════════════════════════════════════════════════════════════

# Start connected to Temporal Cloud (reads from .env.cloud)
[group('cloud')]
up-cloud:
    #!/usr/bin/env sh
    if [ ! -f .env.cloud ]; then
        echo "Error: .env.cloud not found"
        echo ""
        echo "To set up Temporal Cloud:"
        echo "  1. cp .env.cloud.example .env.cloud"
        echo "  2. Edit .env.cloud with your Temporal Cloud credentials"
        echo "  3. For mTLS: place your certificates in ./certs/"
        echo "  4. Run: just up-cloud"
        exit 1
    fi
    docker compose --env-file .env.cloud up -d worker api codec-server frontend
    echo ""
    echo "Connected to TEMPORAL CLOUD"
    echo "  Frontend:     http://localhost:3000"
    echo "  API:          http://localhost:7070"
    echo "  Codec Server: http://localhost:8081"
    echo ""
    echo "View workflows at: https://cloud.temporal.io"
    echo ""

# Start Temporal Cloud with encryption enabled
[group('cloud')]
up-cloud-encrypted:
    #!/usr/bin/env sh
    if [ ! -f .env.cloud ]; then
        echo "Error: .env.cloud not found. Run: just cloud-setup"
        exit 1
    fi
    ENCRYPT_PAYLOADS=true docker compose --env-file .env.cloud up -d worker api codec-server frontend
    echo ""
    echo "Connected to TEMPORAL CLOUD with ENCRYPTION ENABLED"
    echo "  Frontend:     http://localhost:3000"
    echo "  API:          http://localhost:7070"
    echo "  Codec Server: http://localhost:8081"
    echo ""

# Show cloud setup instructions
[group('cloud')]
cloud-setup:
    #!/usr/bin/env sh
    echo "Temporal Cloud Setup"
    echo "===================="
    echo ""
    if [ -f .env.cloud ]; then
        echo "Status: .env.cloud exists"
        echo ""
    else
        echo "Status: .env.cloud not found"
        echo ""
        echo "Step 1: Create config file"
        echo "  cp .env.cloud.example .env.cloud"
        echo ""
    fi
    echo "Step 2: Edit .env.cloud with your Temporal Cloud credentials"
    echo "  - TEMPORAL_ADDRESS: your-namespace.your-account.tmprl.cloud:7233"
    echo "  - TEMPORAL_NAMESPACE: your-namespace.your-account"
    echo ""
    echo "Step 3: Set up authentication (choose one):"
    echo ""
    echo "  Option A - mTLS (recommended for demos):"
    echo "    1. Download certificates from https://cloud.temporal.io"
    echo "    2. mkdir -p certs && cp client.pem client.key certs/"
    echo "    3. Set in .env.cloud:"
    echo "       TEMPORAL_CERT_PATH=/certs/client.pem"
    echo "       TEMPORAL_KEY_PATH=/certs/client.key"
    echo ""
    echo "  Option B - API Key:"
    echo "    1. Create API key at https://cloud.temporal.io/settings/api-keys"
    echo "    2. Set in .env.cloud:"
    echo "       TEMPORAL_API_KEY=your-api-key"
    echo ""
    echo "Step 4: Start the demo"
    echo "  just up-cloud"
    echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# LOGS
# ═══════════════════════════════════════════════════════════════════════════════

# Tail logs: `just logs` or `just logs <worker|api|frontend|temporal|codec-server>`
[group('logs')]
logs *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        docker compose logs -f
    else
        docker compose logs -f {{service}}
    fi

# ═══════════════════════════════════════════════════════════════════════════════
# BUILD
# ═══════════════════════════════════════════════════════════════════════════════

# Build: `just build` or `just build <worker|api|frontend|codec-server>`
[group('build')]
build *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        docker compose build
    else
        docker compose build {{service}}
    fi

# Rebuild (no cache): `just rebuild` or `just rebuild <service>`
[group('build')]
rebuild *service:
    #!/usr/bin/env sh
    if [ -z "{{service}}" ]; then
        docker compose build --no-cache
    else
        docker compose build --no-cache {{service}}
    fi

# ═══════════════════════════════════════════════════════════════════════════════
# CLEANUP
# ═══════════════════════════════════════════════════════════════════════════════

# Stop and remove containers, networks
[group('cleanup')]
clean:
    docker compose down

# Full cleanup: containers, volumes, and unused images
[group('cleanup')]
nuke:
    docker compose down -v
    docker image prune -f
    @echo "Cleaned up containers, volumes, and unused images"
