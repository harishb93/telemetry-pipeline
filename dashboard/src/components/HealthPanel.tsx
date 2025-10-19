import { useEffect, useState } from 'react';
import { AlertCircle, CheckCircle, Clock, Server, Loader2 } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { healthAPI, type ServiceHealthStatus } from '@/api/health';
import { POLLING_INTERVALS } from '@/lib/config';

interface HealthCardProps {
  service: ServiceHealthStatus;
}

function HealthCard({ service }: HealthCardProps) {
  const getStatusConfig = (status: ServiceHealthStatus['status']) => {
    switch (status) {
      case 'ok':
        return {
          icon: CheckCircle,
          badgeVariant: 'default' as const,
          badgeText: 'Healthy',
          iconColor: 'text-green-500',
          borderColor: 'border-green-200',
          bgColor: 'bg-green-50',
        };
      case 'error':
        return {
          icon: AlertCircle,
          badgeVariant: 'destructive' as const,
          badgeText: 'Error',
          iconColor: 'text-red-500',
          borderColor: 'border-red-200',
          bgColor: 'bg-red-50',
        };
      case 'loading':
        return {
          icon: Loader2,
          badgeVariant: 'secondary' as const,
          badgeText: 'Checking...',
          iconColor: 'text-blue-500',
          borderColor: 'border-blue-200',
          bgColor: 'bg-blue-50',
        };
      default:
        return {
          icon: Server,
          badgeVariant: 'outline' as const,
          badgeText: 'Unknown',
          iconColor: 'text-gray-500',
          borderColor: 'border-gray-200',
          bgColor: 'bg-gray-50',
        };
    }
  };

  const config = getStatusConfig(service.status);
  const IconComponent = config.icon;

  return (
    <Card className={`${config.borderColor} ${config.bgColor} transition-colors duration-200`}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">{service.name}</CardTitle>
          <IconComponent 
            className={`h-4 w-4 ${config.iconColor} ${service.status === 'loading' ? 'animate-spin' : ''}`} 
          />
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-center space-x-2 mb-2">
          <Badge variant={config.badgeVariant}>{config.badgeText}</Badge>
        </div>
        
        {service.lastChecked && (
          <div className="flex items-center space-x-1 text-xs text-gray-500">
            <Clock className="h-3 w-3" />
            <span>
              Last checked: {new Date(service.lastChecked).toLocaleTimeString()}
            </span>
          </div>
        )}
        
        {service.error && (
          <CardDescription className="mt-2 text-red-600 text-xs">
            {service.error}
          </CardDescription>
        )}
      </CardContent>
    </Card>
  );
}

interface HealthPanelProps {
  className?: string;
}

export function HealthPanel({ className }: HealthPanelProps) {
  const [services, setServices] = useState<ServiceHealthStatus[]>([
    { name: 'API Gateway', status: 'loading' },
    { name: 'Telemetry Collector', status: 'loading' },
    { name: 'MQ Service', status: 'loading' },
  ]);
  const [isPolling, setIsPolling] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<string>('');

  const fetchHealthStatus = async () => {
    try {
      // Set loading state for all services
      setServices(prev => prev.map(service => ({ ...service, status: 'loading' as const })));
      
      const healthStatuses = await healthAPI.getAllServicesHealth();
      setServices(healthStatuses);
      setLastUpdate(new Date().toISOString());
    } catch (error) {
      console.error('Failed to fetch health statuses:', error);
      
      // Set error state for all services
      setServices(prev => prev.map(service => ({
        ...service,
        status: 'error' as const,
        error: 'Failed to check service health',
        lastChecked: new Date().toISOString(),
      })));
    }
  };

  // Initial fetch and polling setup
  useEffect(() => {
    let intervalId: number | null = null;

    const startPolling = () => {
      // Immediate fetch
      fetchHealthStatus();
      
      // Set up polling
      intervalId = setInterval(fetchHealthStatus, POLLING_INTERVALS.HEALTH) as unknown as number;
    };

    if (isPolling) {
      startPolling();
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [isPolling]);

  // Calculate overall health status
  const overallStatus = services.every(s => s.status === 'ok') 
    ? 'healthy' 
    : services.some(s => s.status === 'error') 
    ? 'degraded' 
    : 'checking';

  const healthyCount = services.filter(s => s.status === 'ok').length;
  const totalCount = services.length;

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Service Health</h2>
          <p className="text-sm text-gray-600">
            {healthyCount}/{totalCount} services healthy
            {lastUpdate && (
              <span className="ml-2">
                • Updated {new Date(lastUpdate).toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <Badge 
            variant={overallStatus === 'healthy' ? 'default' : overallStatus === 'degraded' ? 'destructive' : 'secondary'}
          >
            {overallStatus === 'healthy' ? 'All Systems Operational' : 
             overallStatus === 'degraded' ? 'Service Degraded' : 'Checking...'}
          </Badge>
          
          <button
            onClick={() => setIsPolling(!isPolling)}
            className="text-xs px-2 py-1 rounded border hover:bg-gray-50 transition-colors"
          >
            {isPolling ? 'Pause' : 'Resume'}
          </button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {services.map((service) => (
          <HealthCard key={service.name} service={service} />
        ))}
      </div>
    </div>
  );
}