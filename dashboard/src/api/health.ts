import { API_BASE_URL, COLLECTOR_BASE_URL, MQ_BASE_URL, ENDPOINTS } from '@/lib/config';

export interface HealthResponse {
  status: 'ok' | 'error' | 'healthy';
  timestamp?: string;
  service?: string;
  version?: string;
}

export interface ServiceHealthStatus {
  name: string;
  status: 'ok' | 'error' | 'loading' | 'unknown';
  lastChecked?: string;
  error?: string;
}

class HealthAPI {
  private async fetchHealthWithTimeout(
    url: string,
    serviceName: string,
    timeout = 5000
  ): Promise<HealthResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      
      // Normalize different response formats
      if (data.status === 'healthy') {
        return { ...data, status: 'ok' };
      }
      
      return data;
    } catch (error) {
      clearTimeout(timeoutId);
      console.error(`Health check failed for ${serviceName}:`, error);
      
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new Error(`Timeout: ${serviceName} did not respond within ${timeout}ms`);
        }
        throw error;
      }
      
      throw new Error(`Unknown error checking ${serviceName} health`);
    }
  }

  async getApiGatewayHealth(): Promise<HealthResponse> {
    // For Docker deployment, use direct /health path, otherwise use API_BASE_URL
    const isDockerDeployment = API_BASE_URL.startsWith('/');
    const healthUrl = isDockerDeployment ? ENDPOINTS.HEALTH : `${API_BASE_URL}${ENDPOINTS.HEALTH}`;
    
    return this.fetchHealthWithTimeout(
      healthUrl,
      'API Gateway'
    );
  }

  async getCollectorHealth(): Promise<HealthResponse> {
    return this.fetchHealthWithTimeout(
      `${COLLECTOR_BASE_URL}${ENDPOINTS.COLLECTOR_HEALTH}`,
      'Telemetry Collector'
    );
  }

  async getMQServiceHealth(): Promise<HealthResponse> {
    return this.fetchHealthWithTimeout(
      `${MQ_BASE_URL}${ENDPOINTS.MQ_HEALTH}`,
      'MQ Service'
    );
  }

  async getAllServicesHealth(): Promise<ServiceHealthStatus[]> {
    const services = [
      { name: 'API Gateway', fetchFn: () => this.getApiGatewayHealth() },
      { name: 'Telemetry Collector', fetchFn: () => this.getCollectorHealth() },
      { name: 'MQ Service', fetchFn: () => this.getMQServiceHealth() },
    ];

    const results = await Promise.allSettled(
      services.map(async (service) => {
        try {
          const health = await service.fetchFn();
          return {
            name: service.name,
            status: health.status === 'healthy' ? 'ok' : health.status,
            lastChecked: new Date().toISOString(),
          } as ServiceHealthStatus;
        } catch (error) {
          return {
            name: service.name,
            status: 'error' as const,
            lastChecked: new Date().toISOString(),
            error: error instanceof Error ? error.message : 'Unknown error',
          } as ServiceHealthStatus;
        }
      })
    );

    return results.map((result, index) => {
      if (result.status === 'fulfilled') {
        return result.value;
      } else {
        return {
          name: services[index].name,
          status: 'error' as const,
          lastChecked: new Date().toISOString(),
          error: result.reason?.message || 'Failed to check health',
        };
      }
    });
  }
}

export const healthAPI = new HealthAPI();