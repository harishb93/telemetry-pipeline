// Health Panel Testing Guide
// This file demonstrates how to test the HealthPanel component

/*
## Testing the HealthPanel Component

### 1. Start the backend services:
```bash
# Terminal 1: Start MQ Service
./bin/mq-service --http-port=9090 --grpc-port=9091

# Terminal 2: Start Telemetry Collector  
./bin/telemetry-collector --workers=2 --data-dir=./data --health-port=8080

# Terminal 3: Start API Gateway
COLLECTOR_URL=http://localhost:8080 ./bin/api-gateway --port=8081
```

### 2. Start the dashboard:
```bash
cd dashboard
npm run dev
```

### 3. Expected Health Panel Behavior:

#### When All Services Running:
- API Gateway: Green card with "Healthy" badge
- Telemetry Collector: Green card with "Healthy" badge  
- MQ Service: Green card with "Healthy" badge
- Overall Status: "All Systems Operational" (green badge)

#### When Services are Down:
- Each unavailable service shows: Red card with "Error" badge
- Error message displays connection failure details
- Overall Status: "Service Degraded" (red badge)

#### Loading States:
- During health checks: Blue card with spinning loader
- Badge shows "Checking..." with loading animation

### 4. API Endpoints Tested:
- GET http://localhost:8081/health (API Gateway)
- GET http://localhost:8080/health (Telemetry Collector)  
- GET http://localhost:9090/health (MQ Service)

### 5. Polling Behavior:
- Health checks every 10 seconds (configurable in config.ts)
- Can pause/resume polling with button
- Shows last update timestamp
- Service count: "X/3 services healthy"

### 6. Error Handling:
- Network timeouts (5 second limit)
- HTTP errors (non-200 responses)
- JSON parsing errors
- Service unavailable scenarios

### 7. UI Features:
- Color-coded status cards (green/red/blue/gray)
- Animated loading spinners
- Responsive grid layout
- Real-time status updates
- Manual pause/resume control
*/

export const HEALTH_PANEL_TESTING_NOTES = {
  component: 'HealthPanel',
  location: '/dashboard/src/components/HealthPanel.tsx',
  apiHelper: '/dashboard/src/api/health.ts',
  pollingInterval: '10 seconds',
  timeout: '5 seconds',
  services: [
    { name: 'API Gateway', port: 8081, endpoint: '/health' },
    { name: 'Telemetry Collector', port: 8080, endpoint: '/health' },
    { name: 'MQ Service', port: 9090, endpoint: '/health' }
  ]
};