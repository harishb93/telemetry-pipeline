// Telemetry data types
export interface TelemetryMetrics {
  temperature: number;
  utilization: number;
  memory_used: number;
  power_draw: number;
  fan_speed: number;
}

export interface TelemetryData {
  gpu_id: string;
  hostname: string;
  timestamp: string;
  metrics: TelemetryMetrics;
}

export interface TelemetryResponse {
  data: TelemetryData[] | null;
  total: number;
  pagination: {
    limit: number;
    offset: number;
    has_next: boolean;
  };
}

// Health status types
export interface ServiceHealth {
  status: 'healthy' | 'unhealthy';
  timestamp?: string;
}

export interface ApiGatewayHealth extends ServiceHealth {
  service: string;
  version: string;
  collector: ServiceHealth;
}

// MQ Stats types
export interface TopicStats {
  queue_size: number;
  subscriber_count: number;
  pending_messages: number;
}

export interface MQStats {
  topics: Record<string, TopicStats>;
}

// API error types
export interface ApiError {
  message: string;
  code?: string;
  status?: number;
}