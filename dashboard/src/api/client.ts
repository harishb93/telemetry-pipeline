import type {
  TelemetryResponse,
  ApiGatewayHealth,
  ServiceHealth,
  MQStats,
  ApiError,
} from '@/lib/types';
import { API_BASE_URL, COLLECTOR_BASE_URL, MQ_BASE_URL, ENDPOINTS } from '@/lib/config';

class ApiClient {
  private async fetchWithTimeout(
    url: string,
    options: RequestInit = {},
    timeout = 10000
  ): Promise<Response> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      return response;
    } catch (error) {
      clearTimeout(timeoutId);
      throw error;
    }
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const error: ApiError = {
        message: `HTTP ${response.status}: ${response.statusText}`,
        status: response.status,
      };
      throw new Error(JSON.stringify(error));
    }

    try {
      return await response.json();
    } catch (error) {
      throw new Error('Failed to parse JSON response');
    }
  }

  // API Gateway endpoints
  async getGpus(): Promise<string[]> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}${ENDPOINTS.GPUS}`);
    return this.handleResponse<string[]>(response);
  }

  async getHosts(): Promise<string[]> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}${ENDPOINTS.HOSTS}`);
    return this.handleResponse<string[]>(response);
  }

  async getTelemetry(
    gpuId: string,
    params?: {
      limit?: number;
      offset?: number;
      start?: string;
      end?: string;
    }
  ): Promise<TelemetryResponse> {
    const url = new URL(`${API_BASE_URL}${ENDPOINTS.TELEMETRY(gpuId)}`);
    
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value));
        }
      });
    }

    const response = await this.fetchWithTimeout(url.toString());
    return this.handleResponse<TelemetryResponse>(response);
  }

  async getApiGatewayHealth(): Promise<ApiGatewayHealth> {
    const response = await this.fetchWithTimeout(`${API_BASE_URL}${ENDPOINTS.HEALTH}`);
    return this.handleResponse<ApiGatewayHealth>(response);
  }

  // Collector endpoints
  async getCollectorHealth(): Promise<ServiceHealth> {
    const response = await this.fetchWithTimeout(`${COLLECTOR_BASE_URL}${ENDPOINTS.COLLECTOR_HEALTH}`);
    return this.handleResponse<ServiceHealth>(response);
  }

  // MQ Service endpoints
  async getMQHealth(): Promise<ServiceHealth> {
    const response = await this.fetchWithTimeout(`${MQ_BASE_URL}${ENDPOINTS.MQ_HEALTH}`);
    return this.handleResponse<ServiceHealth>(response);
  }

  async getMQStats(): Promise<MQStats> {
    const response = await this.fetchWithTimeout(`${MQ_BASE_URL}${ENDPOINTS.STATS}`);
    return this.handleResponse<MQStats>(response);
  }
}

export const apiClient = new ApiClient();