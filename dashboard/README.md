# GPU Telemetry Dashboard

A modern React-based dashboard for monitoring GPU telemetry data in real-time.

## Tech Stack

- **React 19** with TypeScript
- **Vite** for fast development and building
- **Tailwind CSS** for styling
- **Recharts** for data visualization
- **Lucide React** for icons

## Features

- Real-time GPU telemetry monitoring
- Service health status tracking
- Interactive charts and graphs
- Message queue statistics
- Responsive design

## Directory Structure

```
src/
├── components/          # UI building blocks
│   ├── ui/             # Basic UI components (Card, Badge, etc.)
│   └── DashboardLayout.tsx  # Layout components
├── pages/              # Main application pages
│   └── Dashboard.tsx   # Main dashboard page
├── api/                # API clients and data fetching
│   └── client.ts       # API client implementation
├── lib/                # Shared utilities
│   ├── config.ts       # Configuration and constants
│   ├── types.ts        # TypeScript type definitions
│   ├── usePolling.ts   # Custom polling hook
│   └── utils.ts        # Utility functions
└── main.tsx           # Application entry point
```

## Development

### Prerequisites

- Node.js 18+ 
- npm or yarn

### Getting Started

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the development server:
   ```bash
   npm run dev
   ```

3. Open http://localhost:5173 in your browser

### Build for Production

```bash
npm run build
```

### Environment Variables

Create a `.env` file in the root directory:

```env
VITE_API_BASE_URL=http://localhost:8081
VITE_COLLECTOR_BASE_URL=http://localhost:8080
VITE_MQ_BASE_URL=http://localhost:9090
```

## API Integration

The dashboard connects to three backend services:

1. **API Gateway** (port 8081) - Main data API
2. **Telemetry Collector** (port 8080) - Health monitoring
3. **MQ Service** (port 9090) - Queue statistics

## Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## Data Flow

1. Fetch GPU list and select GPU for monitoring
2. Poll telemetry data every 5 seconds
3. Display real-time charts with latest metrics
4. Monitor service health status
5. Show message queue statistics
