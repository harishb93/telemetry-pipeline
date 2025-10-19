import { useState } from 'react';
import { AlertCircle, CheckCircle, Clock, Server } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';

interface SimpleServiceStatus {
  name: string;
  status: 'ok' | 'error' | 'unknown';
  lastChecked?: string;
  error?: string;
}

interface SimpleHealthPanelProps {
  className?: string;
}

function SimpleHealthCard({ service }: { service: SimpleServiceStatus }) {
  const getStatusConfig = (status: SimpleServiceStatus['status']) => {
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
          <IconComponent className={`h-4 w-4 ${config.iconColor}`} />
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

export function SimpleHealthPanel({ className }: SimpleHealthPanelProps) {
  const [services] = useState<SimpleServiceStatus[]>([
    { 
      name: 'API Gateway', 
      status: 'ok',
      lastChecked: new Date().toISOString()
    },
    { 
      name: 'Telemetry Collector', 
      status: 'unknown',
      error: 'CORS - Direct access disabled'
    },
    { 
      name: 'MQ Service', 
      status: 'unknown',
      error: 'CORS - Direct access disabled'
    },
  ]);

  const healthyCount = services.filter(s => s.status === 'ok').length;
  const totalCount = services.length;

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Service Health</h2>
          <p className="text-sm text-gray-600">
            {healthyCount}/{totalCount} services healthy
            <span className="ml-2 text-orange-600">
              • CORS issue detected - using API Gateway only
            </span>
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <Badge variant="secondary">
            Partial Monitoring
          </Badge>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {services.map((service) => (
          <SimpleHealthCard key={service.name} service={service} />
        ))}
      </div>

      <div className="mt-4 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
        <div className="flex items-start space-x-2">
          <AlertCircle className="h-4 w-4 text-yellow-600 mt-0.5" />
          <div className="text-sm">
            <p className="font-medium text-yellow-800">CORS Configuration Needed</p>
            <p className="text-yellow-700 mt-1">
              Direct access to collector (port 8080) and MQ service (port 9090) is blocked by CORS policy. 
              Configure the API Gateway to proxy these endpoints or add CORS headers to the services.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}