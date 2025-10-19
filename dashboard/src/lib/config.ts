// API configuration - Use environment variables for flexible deployment
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081';
export const COLLECTOR_BASE_URL = import.meta.env.VITE_COLLECTOR_BASE_URL || 'http://localhost:8080';
export const MQ_BASE_URL = import.meta.env.VITE_MQ_BASE_URL || 'http://localhost:9090';

// Polling intervals (in milliseconds)
export const POLLING_INTERVALS = {
  TELEMETRY: 5000, // 5 seconds
  HEALTH: 10000, // 10 seconds
  STATS: 5000, // 5 seconds
  DASHBOARD: 10000, // 10 seconds - for GPU/host list updates
};

// API endpoints - Dynamic based on environment
const isDockerDeployment = API_BASE_URL.startsWith('/');

export const ENDPOINTS = {
  // GPU and Host endpoints
  GPUS: isDockerDeployment ? '/v1/gpus' : '/api/v1/gpus',
  HOSTS: isDockerDeployment ? '/v1/hosts' : '/api/v1/hosts',
  TELEMETRY: (gpuId: string) => isDockerDeployment ? `/v1/gpus/${gpuId}/telemetry` : `/api/v1/gpus/${gpuId}/telemetry`,
  
  // Health endpoints - Fixed for Docker deployment
  HEALTH: isDockerDeployment ? '/health' : '/health', // Direct /health for both environments
  COLLECTOR_HEALTH: '/health', // Collector health (via COLLECTOR_BASE_URL)
  MQ_HEALTH: '/health', // MQ health (via MQ_BASE_URL) 
  STATS: '/stats', // MQ stats (via MQ_BASE_URL)
} as const;