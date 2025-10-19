import { Activity, AlertCircle, CheckCircle, Server } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

interface HeaderProps {
  title: string;
  subtitle?: string;
}

export function Header({ title, subtitle }: HeaderProps) {
  return (
    <div className="border-b">
      <div className="flex h-16 items-center px-4">
        <div className="flex items-center space-x-2">
          <Activity className="h-6 w-6" />
          <div>
            <h1 className="text-xl font-semibold">{title}</h1>
            {subtitle && (
              <p className="text-sm text-gray-600">{subtitle}</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

interface ServiceStatusProps {
  name: string;
  status: 'healthy' | 'unhealthy' | 'unknown';
  lastUpdated?: string;
}

function ServiceStatus({ name, status, lastUpdated }: ServiceStatusProps) {
  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'healthy':
        return {
          icon: CheckCircle,
          variant: 'default' as const,
          text: 'Healthy',
          color: 'text-green-500',
        };
      case 'unhealthy':
        return {
          icon: AlertCircle,
          variant: 'destructive' as const,
          text: 'Unhealthy',
          color: 'text-red-500',
        };
      default:
        return {
          icon: Server,
          variant: 'secondary' as const,
          text: 'Unknown',
          color: 'text-gray-500',
        };
    }
  };

  const config = getStatusConfig(status);
  const IconComponent = config.icon;

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{name}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center space-x-2">
          <IconComponent className={`h-4 w-4 ${config.color}`} />
          <Badge variant={config.variant}>{config.text}</Badge>
        </div>
        {lastUpdated && (
          <CardDescription className="mt-2">
            Last updated: {new Date(lastUpdated).toLocaleTimeString()}
          </CardDescription>
        )}
      </CardContent>
    </Card>
  );
}

interface DashboardGridProps {
  services: Array<{
    name: string;
    status: 'healthy' | 'unhealthy' | 'unknown';
    lastUpdated?: string;
  }>;
  children?: React.ReactNode;
}

export function DashboardGrid({ services, children }: DashboardGridProps) {
  return (
    <div className="flex-1 space-y-4 p-4 pt-6 md:p-8">
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {services.map((service) => (
          <ServiceStatus
            key={service.name}
            name={service.name}
            status={service.status}
            lastUpdated={service.lastUpdated}
          />
        ))}
      </div>
      {children && (
        <div className="space-y-4">
          {children}
        </div>
      )}
    </div>
  );
}