# GitHub Event System

A real-time GitHub webhook event processing system with retry logic, dead letter queues, and a beautiful dashboard for visualizing event pipelines.

## Architecture Overview

This system demonstrates a production-grade event processing pipeline with the following components:

- **API Service**: Receives GitHub webhooks and queues events for processing
- **Worker Service**: Processes events from Redis queues with retry logic
- **PostgreSQL**: Stores event data and processing status
- **Redis**: Provides queuing, caching, and deduplication
- **Frontend**: Real-time dashboard for monitoring event flow

## System Concepts

### Event Pipeline Flow
1. **GitHub Webhook** -> API Service receives HTTP POST
2. **API Service** -> Validates payload and pushes to Redis queue
3. **Redis Queue** -> Holds events for workers to process
4. **Worker Pool** -> Concurrently processes events
5. **Database** -> Stores final event status and metadata
6. **Frontend** -> Real-time visualization of event flow

### Key Features
- **Retry Logic**: Failed events are retried with exponential backoff
- **Dead Letter Queue**: Events that fail after max retries are moved to DLQ
- **Idempotency**: Duplicate events are detected and skipped
- **Concurrent Processing**: Multiple workers process events in parallel
- **Production-Grade Observability**: Complete event lifecycle logging and monitoring
- **Real-time Dashboard**: Live visualization of event pipeline status with detailed logs

## Quick Start (Docker Compose)

### Prerequisites
- Docker and Docker Compose installed
- Git (for testing webhooks)

### Running the System

1. **Clone and setup:**
```bash
git clone <repository-url>
cd github-event-system
```

2. **Start all services:**
```bash
docker-compose up --build
```

3. **Access the dashboard:**
- Frontend Dashboard: http://localhost:3000
- API Health Check: http://localhost:8080/events
- Observability Dashboard: http://localhost:3000 (with logs and filtering)

### Testing Webhooks

1. **Create a test webhook** (using ngrok for local testing):
```bash
# Install ngrok if needed
npm install -g ngrok

# Start ngrok to expose local port
ngrok http 8080
```

2. **Configure GitHub webhook:**
- Go to your GitHub repository settings
- Add webhook pointing to `https://your-ngrok-url.ngrok.io/webhook`
- Set content type to `application/json`
- Select "Push" events

3. **Trigger events:**
```bash
# Make a commit to trigger webhook
git add .
git commit -m "Test webhook trigger"
git push origin main
```

## Manual Testing

You can also test the webhook manually:

```bash
# Test webhook endpoint
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: push" \
  -H "X-Github-Delivery: test-123" \
  -d '{
    "repository": {
      "name": "test-repo",
      "full_name": "user/test-repo"
    },
    "ref": "refs/heads/main",
    "pusher": {
      "name": "testuser"
    },
    "commits": [{
      "id": "abc123",
      "message": "Test commit message",
      "timestamp": "2024-01-01T00:00:00Z"
    }],
    "compare": "https://github.com/user/repo/compare/main...main"
  }'
```

## Development Setup

### Running Services Individually

1. **Start dependencies:**
```bash
# Start PostgreSQL and Redis
docker-compose up postgres redis -d
```

2. **Run API service:**
```bash
cd api
go mod tidy
go run main.go
```

3. **Run worker service:**
```bash
cd worker
go mod tidy
go run main.go
```

4. **Run frontend:**
```bash
cd frontend
python3 -m http.server 3000  # or use any static server
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `github_events` | Database name |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `WORKER_COUNT` | `5` | Number of worker goroutines |

## Kubernetes Deployment

### Prerequisites
- Kubernetes cluster (minikube, Docker Desktop, etc.)
- kubectl configured

### Deploy to Kubernetes

1. **Create Redis deployment:**
```bash
kubectl apply -f k8s/redis.yaml
```

2. **Create PostgreSQL deployment:**
```bash
kubectl apply -f k8s/postgres.yaml
```

3. **Build and push images:**
```bash
# Build API image
docker build -t github-api ./api
docker tag github-api your-registry/github-api
docker push your-registry/github-api

# Build Worker image
docker build -t github-worker ./worker
docker tag github-worker your-registry/github-worker
docker push your-registry/github-worker
```

4. **Deploy services:**
```bash
kubectl apply -f k8s/api.yaml
kubectl apply -f k8s/worker.yaml
```

5. **Access services:**
```bash
# Get API service URL
kubectl get service api

# Port-forward for local access
kubectl port-forward service/api 8080:8080
```

## Monitoring and Debugging

### Checking Logs
```bash
# Docker Compose logs
docker-compose logs -f api
docker-compose logs -f worker
docker-compose logs -f postgres
docker-compose logs -f redis

# Kubernetes logs
kubectl logs -f deployment/api
kubectl logs -f deployment/worker
```

### Database Queries
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U postgres -d github_events

# Check event status
SELECT status, COUNT(*) FROM events GROUP BY status;

# Check recent events
SELECT * FROM events ORDER BY created_at DESC LIMIT 10;
```

### Redis Monitoring
```bash
# Connect to Redis
docker-compose exec redis redis-cli

# Check queue lengths
LLEN github_events_queue
LLEN retry_queue
LLEN dead_letter_queue

# Monitor processed events
SCARD processed_events
```

## Event Processing Logic

### Success Case
- Events are processed immediately
- Status set to "success"
- Event marked as processed in Redis

### Retry Logic
- Events from "github-event-system" repo are simulated to fail
- Retry attempts use exponential backoff (1s, 2s, 3s delays)
- Max 3 retry attempts
- Each retry updates the retry count

### Dead Letter Queue
- Events failing after 3 retries are moved to DLQ
- Status set to "failed"
- Manual intervention required to process

### Deduplication
- Events are tracked by delivery ID in Redis set
- Duplicate events are skipped to ensure idempotency

## Observability Dashboard

The system includes comprehensive production-grade observability features:

### Logging System
- **Step-by-step logging**: Every event processing step is logged with timestamps
- **Persistent storage**: Logs stored in PostgreSQL with incremental updates
- **Complete lifecycle tracking**: From webhook receipt to final processing

### Dashboard Features
- **Real-time Statistics**: Total, success, failed, and retry counts
- **Event Timeline**: Visual pipeline flow for each event
- **Event Details**: Repository, branch, commit message, delivery ID
- **Status Indicators**: Color-coded status badges (green=success, red=failed, yellow=retry)
- **Filter Dropdown**: Filter events by status (All/Success/Failed/Retry/Pending)
- **Terminal-style Logs Modal**: Click "View Logs" to see detailed processing timeline
- **Auto-scroll logs**: Automatically scroll to latest log entries
- **Auto-refresh**: Dashboard updates every 5 seconds

### Log Format
Each log entry includes timestamp and detailed message:
```
[2026-04-21 13:23:52] Event received from GitHub
[2026-04-21 13:23:52] Event received and validated
[2026-04-21 13:23:52] Event pushed to Redis queue for processing
[2026-04-21 13:23:52] Worker picked up event for processing
[2026-04-21 13:23:52] Starting initial event processing
[2026-04-21 13:23:52] Successfully processed event: Complete logging test
[2026-04-21 13:23:52] Marking event as processed
[2026-04-21 13:23:52] Event processing completed successfully
```

### API Endpoints
- `GET /events` - List all events with filtering support
- `GET /events/:id/logs` - Retrieve detailed logs for specific event
- `POST /webhook` - Receive GitHub webhook events

### Frontend Dashboard

The dashboard provides:
- **Real-time Statistics**: Total, success, failed, and retry counts
- **Event Timeline**: Visual pipeline flow for each event
- **Event Details**: Repository, branch, commit message, delivery ID
- **Status Indicators**: Color-coded status badges
- **Auto-refresh**: Updates every 5 seconds

### Dashboard Features
- Click any event to see its pipeline journey
- Visual flow from GitHub through API, Redis, Worker, to Database
- Retry stages shown for events with retry attempts
- Timeline showing event processing history
- **NEW**: Click "View Logs" to see detailed processing timeline
- **NEW**: Filter events by status
- **NEW**: Row highlighting based on event status

## Troubleshooting

### Common Issues

1. **API not receiving webhooks:**
   - Check GitHub webhook configuration
   - Verify ngrok tunnel if using local testing
   - Check API service logs

2. **Events not processing:**
   - Check worker service logs
   - Verify Redis connectivity
   - Check worker pool size

3. **Database connection errors:**
   - Verify PostgreSQL is running
   - Check connection parameters
   - Review database logs

4. **Frontend not loading data:**
   - Check API service is accessible
   - Verify CORS configuration
   - Check browser console for errors

### Health Checks
```bash
# API health check
curl http://localhost:8080/events

# Database connectivity
docker-compose exec postgres pg_isready -U postgres

# Redis connectivity
docker-compose exec redis redis-cli ping
```

## Performance Considerations

- **Worker Pool**: Adjust `WORKER_COUNT` based on load
- **Database Indexes**: Added on status, created_at, and repo fields
- **Redis Memory**: Monitor Redis memory usage with high event volumes
- **Connection Pooling**: Consider connection pooling for production

## Security Notes

- **Webhook Secrets**: Implement webhook signature verification in production
- **Database Credentials**: Use environment variables for sensitive data
- **Network Security**: Use TLS for external communications
- **Input Validation**: Add proper input validation for webhook payloads

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## License

MIT License - see LICENSE file for details