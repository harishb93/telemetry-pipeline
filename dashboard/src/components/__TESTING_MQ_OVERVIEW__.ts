// MQ Overview Panel Testing Guide
// This file demonstrates how to test the MQOverview component

/*
## Testing the MQOverview Component

### 1. Start the MQ Service:
```bash
./bin/mq-service --http-port=9090 --grpc-port=9091 --persistence=true
```

### 2. Generate some test data:
```bash
# Send some test messages to create topics
curl -X POST http://localhost:9090/publish \
  -H "Content-Type: application/json" \
  -d '{"topic": "telemetry", "message": "test message 1"}'

curl -X POST http://localhost:9090/publish \
  -H "Content-Type: application/json" \
  -d '{"topic": "alerts", "message": "alert message 1"}'

curl -X POST http://localhost:9090/publish \
  -H "Content-Type: application/json" \
  -d '{"topic": "metrics", "message": "metric message 1"}'
```

### 3. Start the dashboard:
```bash
cd dashboard
npm run dev
```

### 4. Expected MQOverview Behavior:

#### Summary Cards (Top Row):
- **Topics**: Number of active topics
- **Total Messages**: Sum of all queue_size values
- **Pending**: Sum of all pending_messages values  
- **Subscribers**: Sum of all subscriber_count values

#### Topics Overview Chart (Bar Chart):
- X-axis: Topic names
- Y-axis: Message counts
- Three bars per topic:
  - Blue: Queue Size
  - Orange: Pending Messages
  - Green: Subscribers

#### Individual Topic Cards:
- **Topic Name** with database icon
- **Three Metrics**: Queue Size, Pending, Subscribers
- **Mini Pie Chart**: Shows processed vs pending messages
- **Utilization Badge**: Color-coded percentage
  - Green (0-50%): Normal
  - Yellow (50-80%): Warning
  - Red (80%+): Critical

#### Topics Details Table:
- **Responsive table** with topic metrics
- **Color-coded badges** for each metric
- **Status column** showing utilization percentage
- **Icons** for each metric type

### 5. API Integration:
- **Endpoint**: GET http://localhost:9090/stats
- **Polling**: Every 10 seconds (configurable)
- **Timeout**: 10 seconds per request
- **Auto-refresh**: Real-time updates

### 6. Expected /stats Response:
```json
{
  "topics": {
    "telemetry": {
      "queue_size": 50,
      "subscriber_count": 2,
      "pending_messages": 5
    },
    "alerts": {
      "queue_size": 10,
      "subscriber_count": 1,
      "pending_messages": 0
    }
  }
}
```

### 7. Interactive Features:
- **Pause/Resume Button**: Stop/start polling
- **Refresh Button**: Manual refresh with loading spinner
- **Real-time Updates**: Auto-refresh every 10 seconds
- **Timestamp**: Shows last update time

### 8. Error Handling:
- **Network Errors**: Red error card with details
- **Empty State**: "No topics found" message
- **Loading States**: Loading indicators during fetch
- **Graceful Degradation**: Continues to show old data on errors

### 9. Responsive Design:
- **Desktop**: 3-column topic cards, full table
- **Tablet**: 2-column topic cards, scrollable table
- **Mobile**: 1-column layout, horizontal scroll table

### 10. Chart Features:
- **Bar Chart**: Comparative view of all topics
- **Pie Charts**: Individual topic utilization
- **Tooltips**: Hover details for all charts
- **Color Coding**: Consistent color scheme
- **Responsive**: Auto-sizing containers

### 11. Status Indicators:
- **Utilization Badges**:
  - 0-50%: Green (Default)
  - 50-80%: Yellow (Secondary)
  - 80%+: Red (Destructive)
- **Overall Status**: Based on highest utilization
- **Subscriber Status**: Red if no subscribers

### 12. Testing Scenarios:

#### Normal Operation:
- Topics with balanced queues
- Active subscribers
- Low pending messages

#### High Load:
- Large queue sizes
- High pending message counts
- Multiple active topics

#### Error Conditions:
- MQ service down
- Network timeouts
- Malformed responses

#### Edge Cases:
- Empty topics list
- Zero subscribers
- Zero queue sizes
*/

export const MQ_OVERVIEW_TESTING_NOTES = {
  component: 'MQOverview',
  location: '/dashboard/src/components/MQOverview.tsx',
  endpoint: 'http://localhost:9090/stats',
  pollingInterval: '10 seconds',
  features: [
    'Real-time stats monitoring',
    'Responsive table layout',
    'Bar charts for topic comparison', 
    'Pie charts for utilization',
    'Error handling and loading states',
    'Manual refresh controls',
    'Color-coded status indicators'
  ]
};