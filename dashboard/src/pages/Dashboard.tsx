import { useState, useEffect } from 'react';
import { Header } from '@/components/DashboardLayout';
import { HealthPanel } from '@/components/HealthPanel';
import { MQOverview } from '@/components/MQOverview';
import { HostsOverview } from '@/components/HostsOverview';
import { GPUSelection } from '@/components/GPUSelection';
import { ErrorBoundary } from '@/components/ErrorBoundary';
import { apiClient } from '@/api/client';
import { usePolling } from '@/lib/usePolling';
import { POLLING_INTERVALS } from '@/lib/config';

export function Dashboard() {
  const [selectedGpu, setSelectedGpu] = useState<string>('');
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
        </div>
      </div>
    </div>
  );
}