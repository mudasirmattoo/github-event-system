# GitHub Webhook Setup Guide

## Current Status
The GitHub Event System is working correctly! We've verified that:
- API webhook endpoint is accessible at `http://localhost:8081/webhook`
- Worker is processing events from Redis queue
- Events are being stored in PostgreSQL database
- Dashboard displays events in real-time

## Why GitHub Pushes Don't Appear
When you push to GitHub, no new events appear because GitHub webhooks need to be configured to send events to your public-facing endpoint. Currently, your system is running locally.

## Solution Options

### Option 1: Use ngrok (Recommended for Development)
Expose your local server to the internet using ngrok:

1. **Install ngrok:**
   ```bash
   # Download and install ngrok from https://ngrok.com/download
   # Or use package manager:
   # sudo pacman -S ngrok  # Arch Linux
   ```

2. **Start ngrok:**
   ```bash
   ngrok http 8081
   ```

3. **Copy the ngrok URL** (e.g., `https://abc123.ngrok.io`)

4. **Configure GitHub Webhook:**
   - Go to your repository on GitHub
   - Settings > Webhooks > Add webhook
   - Payload URL: `https://abc123.ngrok.io/webhook`
   - Content type: `application/json`
   - Select events: "Pushes", "Pull requests", etc.
   - Add webhook

### Option 2: Deploy to Cloud
Deploy your system to a cloud service with a public IP:
- AWS EC2
- Google Cloud Platform
- DigitalOcean
- Heroku

### Option 3: GitHub Actions Integration
Use GitHub Actions to simulate webhook events for testing.

## Testing Webhook Locally

### Manual Test (Already Working)
```bash
curl -X POST http://localhost:8081/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "zen":"GitHub",
    "hook_id":123456,
    "repository":{"id":789,"name":"github-event-system","full_name":"mudasirmattoo/github-event-system"},
    "sender":{"id":123456,"login":"mudasirmattoo"},
    "pusher":{"name":"mudasirmattoo","email":"mudasirmattoo@example.com"},
    "ref":"refs/heads/main",
    "commits":[{"id":"abc123","message":"Test commit","author":{"name":"Test User","email":"test@example.com"}}],
    "head_commit":{"id":"abc123","message":"Test commit","author":{"name":"Test User","email":"test@example.com"}}
  }'
```

### Check Results
```bash
# View events in dashboard
open http://localhost:3000/events

# Or check via API
curl http://localhost:8081/api/events | jq '.'
```

## Current Docker Setup Status
All services are running correctly:
- **Frontend**: http://localhost:3000/events
- **API**: http://localhost:8081/api/events
- **PostgreSQL**: localhost:5433
- **Redis**: localhost:6380
- **Worker**: Processing events successfully

## Expected Behavior
When webhook is properly configured:
1. GitHub sends webhook to your endpoint
2. API receives and validates webhook
3. API pushes event to Redis queue
4. Worker processes event from queue
5. Worker stores event in PostgreSQL
6. Dashboard shows new event in real-time

## Troubleshooting

### Events Not Appearing
1. Check webhook URL is correct and accessible
2. Verify webhook is active in GitHub repository settings
3. Check Docker services are running: `docker compose ps`
4. Check API logs: `docker compose logs api`
5. Check worker logs: `docker compose logs worker`

### Webhook Delivery Failures
1. Check GitHub webhook delivery logs in repository settings
2. Verify your endpoint returns 200 OK
3. Check network connectivity and firewall settings

### Worker Not Processing
1. Check Redis connection: `docker compose exec worker redis-cli ping`
2. Check database connection: `docker compose exec worker psql -h postgres -U postgres -d github_events`
3. Restart worker: `docker compose restart worker`

## Security Notes
- Use HTTPS for webhook URLs in production
- Validate webhook signatures using GitHub secrets
- Consider IP whitelisting for additional security
- Monitor webhook delivery logs for suspicious activity

## Production Deployment
For production use, consider:
- Load balancer with SSL termination
- Multiple API instances behind load balancer
- Redis cluster for high availability
- PostgreSQL replication
- Monitoring and alerting
- Log aggregation
- Backup strategies
