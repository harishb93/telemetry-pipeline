import { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Header } from '@/components/DashboardLayout';
import { HealthPanel } from '@/components/HealthPanel';
import { MQOverview } from '@/components/MQOverview';
import { HostsOverview } from '@/components/HostsOverview';
import { GPUSelection } from '@/components/GPUSelection';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { apiClient } from '@/api/client';
import { usePolling } from '@/lib/usePolling';
import { POLLING_INTERVALS } from '@/lib/config';
import type { TelemetryData } from '@/lib/types';

interface ChartData {
  timestamp: string;
  temperature: number;
  utilization: number;
  power_draw: number;
  memory_used: number;
}

export function Dashboard() {
  const [selectedGpu, setSelectedGpu] = useState<string>('');
  const [chartData, setChartData] = useState<ChartData[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  console.log('Dashboard: Rendering with:', { selectedGpu, isLoading });



  // Telemetry polling for selected GPU
  const { data: telemetryData } = usePolling(
    () => selectedGpu ? apiClient.getTelemetry(selectedGpu, { limit: 50 }) : Promise.resolve(null),
    { 
      interval: POLLING_INTERVALS.TELEMETRY, 
      enabled: !!selectedGpu 
    }
  );

    // Load initial data
  useEffect(() => {
    const loadInitialData = async () => {
      try {
        console.log('Dashboard: Loading initial data...');
        setIsLoading(false);
      } catch (error) {
        console.error('Dashboard: Error in initial load:', error);
        setIsLoading(false);
      }
    };

    loadInitialData();
  }, []);

  // Update chart data when telemetry data changes
  useEffect(() => {
    if (telemetryData?.data) {
      const newChartData: ChartData[] = telemetryData.data
        .slice(0, 20) // Limit to last 20 points
        .reverse() // Show oldest first
        .map((item: TelemetryData) => ({
          timestamp: new Date(item.timestamp).toLocaleTimeString(),
          temperature: item.metrics.temperature,
          utilization: item.metrics.utilization,
          power_draw: item.metrics.power_draw,
          memory_used: item.metrics.memory_used / 1024, // Convert to GB
        }));
      
      setChartData(newChartData);
    }
  }, [telemetryData]);

  console.log('Dashboard: About to render...');

  if (isLoading) {
    return (
      <div className="flex h-screen flex-col">
        <Header 
          title="GPU Telemetry Dashboard" 
          subtitle="Real-time monitoring and analytics"
        />
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <p className="mt-4 text-gray-600">Loading dashboard...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen flex-col">
      <Header 
        title="GPU Telemetry Dashboard" 
        subtitle="Real-time monitoring and analytics"
      />
      
      <div className="flex-1 space-y-6 p-4 pt-6 md:p-8">
        {/* Health Panel */}
        <ErrorBoundary componentName="HealthPanel">
          <HealthPanel />
        </ErrorBoundary>
        
        {/* MQ Overview Panel */}
        <ErrorBoundary componentName="MQOverview">
          <MQOverview />
        </ErrorBoundary>

        {/* Hosts Overview Panel */}
        <ErrorBoundary componentName="HostsOverview">
          <HostsOverview />
        </ErrorBoundary>
        
        <div className="space-y-4">
        {/* GPU Selection */}
        <ErrorBoundary componentName="GPUSelection">
          <GPUSelection 
            selectedGpu={selectedGpu}
            onGpuSelect={setSelectedGpu}
            telemetryDataPoints={telemetryData?.total || 0}
          />
        </ErrorBoundary>

        {/* Telemetry Chart */}
        {chartData.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle>Real-time Metrics - {selectedGpu}</CardTitle>
              <CardDescription>
                Live telemetry data from the last {chartData.length} measurements
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="h-[400px]">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="timestamp" />
                    <YAxis />
                    <Tooltip />
                    <Legend />
                    <Line 
                      type="monotone" 
                      dataKey="temperature" 
                      stroke="#8884d8" 
                      name="Temperature (°C)"
                    />
                    <Line 
                      type="monotone" 
                      dataKey="utilization" 
                      stroke="#82ca9d" 
                      name="Utilization (%)"
                    />
                    <Line 
                      type="monotone" 
                      dataKey="power_draw" 
                      stroke="#ffc658" 
                      name="Power (W)"
                    />
                    <Line 
                      type="monotone" 
                      dataKey="memory_used" 
                      stroke="#ff7300" 
                      name="Memory (GB)"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>
        )}


        </div>
      </div>
    </div>
  );
}