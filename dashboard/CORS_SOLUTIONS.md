# CORS Solutions for Telemetry Dashboard

The dashboard is experiencing CORS (Cross-Origin Resource Sharing) issues when trying to access the collector and MQ service directly from the browser.

## Problem
- ✅ API Gateway (port 8081) works - has CORS configured
- ❌ Telemetry Collector (port 8080) - CORS blocked  
- ❌ MQ Service (port 9090) - CORS blocked

## Solution 1: Add CORS Headers to Backend Services (Recommended)

### For Go Services (Collector & MQ):

Add CORS middleware to each service:

```go
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

// Apply to your HTTP server
mux := http.NewServeMux()
// ... add your routes
server := &http.Server{
    Addr:    ":8080", // or :9090 for MQ
    Handler: corsMiddleware(mux),
}
```

### For Development Only:
```go
w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
```

### For Production:
```go
w.Header().Set("Access-Control-Allow-Origin", "https://yourdomain.com")
```

## Solution 2: Proxy Through API Gateway

Add proxy endpoints to the API Gateway:

```go
// In your API Gateway
func setupProxyRoutes(mux *http.ServeMux) {
    // Proxy to collector health
    mux.HandleFunc("/collector/health", func(w http.ResponseWriter, r *http.Request) {
        proxyRequest(w, r, "http://localhost:8080/health")
    })
    
    // Proxy to MQ health  
    mux.HandleFunc("/mq/health", func(w http.ResponseWriter, r *http.Request) {
        proxyRequest(w, r, "http://localhost:9090/health")
    })
    
    // Proxy to MQ stats
    mux.HandleFunc("/mq/stats", func(w http.ResponseWriter, r *http.Request) {
        proxyRequest(w, r, "http://localhost:9090/stats")
    })
}

func proxyRequest(w http.ResponseWriter, r *http.Request, target string) {
    resp, err := http.Get(target)
    if err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    defer resp.Body.Close()
    
    // Copy headers
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}
```

## Solution 3: Development Proxy (Vite)

Add proxy to vite.config.ts:

```typescript
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/collector': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/collector/, '')
      },
      '/mq': {
        target: 'http://localhost:9090', 
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/mq/, '')
      }
    }
  }
})
```

## Current Dashboard Status

The dashboard now uses simplified components that:
- ✅ Show API Gateway status (working)
- ⚠️ Display CORS warnings for other services
- ✅ Still show GPU telemetry data (works via API Gateway)
- ⚠️ Placeholder for MQ stats until CORS is fixed

## Quick Fix Commands

### Start services with CORS (if supported):
```bash
# If your services support CORS flags:
./bin/telemetry-collector --cors-origin="http://localhost:5173"
./bin/mq-service --cors-origin="http://localhost:5173"
```

### Test endpoints directly:
```bash
# These should work from terminal but fail from browser:
curl http://localhost:8080/health
curl http://localhost:9090/health 
curl http://localhost:9090/stats
```

## Restore Full Dashboard

Once CORS is fixed, restore the full components:

```typescript
// In Dashboard.tsx
import { HealthPanel } from '@/components/HealthPanel';
import { MQOverview } from '@/components/MQOverview';

// Replace simplified components
<HealthPanel />
<MQOverview />
```

## Testing CORS Fix

1. Open browser dev tools (F12)
2. Check console for CORS errors
3. Network tab should show successful requests
4. Health cards should turn green
5. MQ stats should populate

The dashboard is now rendering and will work fully once CORS headers are added to the backend services!