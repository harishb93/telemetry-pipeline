import { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Header } from '@/components/DashboardLayout';
import { HealthPanel } from '@/components/HealthPanel';
import { MQOverview } from '@/components/MQOverview';
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

  // Fetch GPU list once
  const [gpus, setGpus] = useState<string[]>([]);
  const [hosts, setHosts] = useState<string[]>([]);



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
        const [gpuList, hostList] = await Promise.all([
          apiClient.getGpus(),
          apiClient.getHosts(),
        ]);
        setGpus(gpuList);
        setHosts(hostList);
        
        // Select first GPU by default
        if (gpuList.length > 0 && !selectedGpu) {
          setSelectedGpu(gpuList[0]);
        }
      } catch (error) {
        console.error('Failed to load initial data:', error);
      }
    };

    loadInitialData();
  }, [selectedGpu]);

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

  return (
    <div className="flex h-screen flex-col">
      <Header 
        title="GPU Telemetry Dashboard" 
        subtitle="Real-time monitoring and analytics"
      />
      
      <div className="flex-1 space-y-6 p-4 pt-6 md:p-8">
        {/* Health Panel */}
        <HealthPanel />
        
        {/* MQ Overview Panel */}
        <MQOverview />
        
        <div className="space-y-4">
        {/* GPU Selection */}
        <Card>
          <CardHeader>
            <CardTitle>GPU Selection</CardTitle>
            <CardDescription>
              {gpus.length} GPUs available, {hosts.length} hosts
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {gpus.map((gpu) => (
                <button
                  key={gpu}
                  onClick={() => setSelectedGpu(gpu)}
                  className={`px-3 py-1 rounded text-sm border transition-colors ${
                    selectedGpu === gpu
                      ? 'bg-blue-600 text-white border-blue-600'
                      : 'bg-white hover:bg-gray-50 hover:text-gray-900 border-gray-200'
                  }`}
                >
                  {gpu}
                </button>
              ))}
            </div>
            {selectedGpu && (
              <p className="text-sm text-gray-600 mt-2">
                Selected: {selectedGpu} • {telemetryData?.total || 0} data points
              </p>
            )}
          </CardContent>
        </Card>

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