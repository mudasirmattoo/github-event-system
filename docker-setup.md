# GitHub Event System - Docker Setup

## Overview
This Docker Compose setup includes all services needed to run the GitHub Event System:
- **API Service**: Handles GitHub webhooks and serves the frontend
- **Worker Service**: Processes events from Redis queue with retry logic
- **Frontend**: Nginx serving the web UI
- **PostgreSQL**: Database for event storage
- **Redis**: Cache and message queue

## Quick Start

### Prerequisites
- Docker and Docker Compose installed
- Git (for cloning the repository)

### Running the System

1. **Start all services:**
   ```bash
   docker-compose up -d
   ```

2. **Check service status:**
   ```bash
   docker-compose ps
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f
   ```

4. **Access the UI:**
   - Frontend: http://localhost:80
   - API: http://localhost:8080
   - Events Dashboard: http://localhost/events

### Stopping the System

```bash
docker-compose down
```

## Services

### API Service
- **Port**: 8080
- **Endpoints**:
  - `GET /events` - Serves the frontend UI
  - `GET /api/events` - Returns event data
  - `POST /webhook` - Receives GitHub webhooks

### Worker Service
- **Replicas**: 2 (configurable via `WORKER_COUNT` env var)
- **Processes**: Events from Redis queues
- **Retry Logic**: 3 attempts with exponential backoff

### Frontend
- **Port**: 80
- **Technology**: Nginx serving static HTML/CSS/JS
- **Features**: Real-time event dashboard, pipeline visualization

### PostgreSQL
- **Port**: 5432
- **Database**: github_events
- **Credentials**: postgres/postgres

### Redis
- **Port**: 6379
- **Uses**: Message queuing and duplicate detection

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | postgres | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_NAME` | github_events | Database name |
| `DB_USER` | postgres | Database user |
| `DB_PASSWORD` | postgres | Database password |
| `REDIS_HOST` | redis | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `WORKER_COUNT` | 5 | Number of worker processes |

### Custom Configuration

Create a `.env` file to override defaults:
```bash
DB_HOST=my-postgres
REDIS_HOST=my-redis
WORKER_COUNT=10
```

## Development

### Building Images

```bash
docker-compose build
```

### Rebuilding Specific Services

```bash
docker-compose build api
docker-compose build worker
docker-compose build frontend
```

### Development Mode

For development with hot reload:
```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up
```

## Monitoring

### Health Checks
All services include health checks:
- PostgreSQL: `pg_isready`
- Redis: `redis-cli ping`
- API: HTTP endpoint check

### Logs

View logs for specific services:
```bash
docker-compose logs -f api
docker-compose logs -f worker
docker-compose logs -f postgres
docker-compose logs -f redis
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 80, 8080, 5432, 6379 are available
2. **Database connection**: Check PostgreSQL health status
3. **Redis connection**: Verify Redis is running and accessible

### Reset Everything

```bash
docker-compose down -v  # Remove volumes
docker-compose up -d   # Start fresh
```

## Architecture

```
GitHub Webhook -> API -> Redis Queue -> Worker -> PostgreSQL
                      |
                      v
                   Frontend <- API
```

### Flow
1. GitHub sends webhook to API
2. API validates and pushes to Redis queue
3. Worker processes events with retry logic
4. Results stored in PostgreSQL
5. Frontend displays real-time data

## Scaling

### Horizontal Scaling
```bash
# Scale workers
docker-compose up -d --scale worker=5
```

### Resource Limits
Adjust resource limits in `docker-compose.yml`:
```yaml
services:
  api:
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
```

## Security

- Internal services communicate via Docker network
- Only frontend exposed to internet
- Environment variables for sensitive data
- Security headers in Nginx

## Backup

### Database Backup
```bash
docker-compose exec postgres pg_dump -U postgres github_events > backup.sql
```

### Restore
```bash
docker-compose exec -T postgres psql -U postgres github_events < backup.sql
```

## Contributing

1. Make changes to code
2. Build and test: `docker-compose build && docker-compose up`
3. Submit pull request

## License

MIT License - see LICENSE file for details
