import { Database, AlertCircle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';

interface SimpleMQOverviewProps {
  className?: string;
}

export function SimpleMQOverview({ className }: SimpleMQOverviewProps) {
  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Message Queue Overview</h2>
          <p className="text-sm text-gray-600">
            Stats temporarily unavailable due to CORS
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <Badge variant="secondary">
            CORS Issue
          </Badge>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4 mb-6">
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center space-x-2">
              <Database className="h-4 w-4 text-blue-500" />
              <div>
                <div className="text-2xl font-bold">--</div>
                <div className="text-xs text-gray-500">Topics</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center space-x-2">
              <Database className="h-4 w-4 text-green-500" />
              <div>
                <div className="text-2xl font-bold">--</div>
                <div className="text-xs text-gray-500">Total Messages</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center space-x-2">
              <Database className="h-4 w-4 text-orange-500" />
              <div>
                <div className="text-2xl font-bold">--</div>
                <div className="text-xs text-gray-500">Pending</div>
              </div>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="pt-4">
            <div className="flex items-center space-x-2">
              <Database className="h-4 w-4 text-purple-500" />
              <div>
                <div className="text-2xl font-bold">--</div>
                <div className="text-xs text-gray-500">Subscribers</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="border-yellow-200 bg-yellow-50">
        <CardHeader>
          <CardTitle className="text-sm font-medium flex items-center space-x-2">
            <AlertCircle className="h-4 w-4 text-yellow-600" />
            <span>MQ Stats Unavailable</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <CardDescription className="text-yellow-700">
            Cannot access MQ service stats due to CORS policy. The MQ service (port 9090) needs to be proxied 
            through the API Gateway or have CORS headers configured to allow browser access.
          </CardDescription>
          <div className="mt-3 text-sm text-yellow-800">
            <p><strong>Solutions:</strong></p>
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>Add CORS headers to MQ service</li>
              <li>Proxy /mq/stats through API Gateway</li>
              <li>Use API Gateway as single entry point</li>
            </ul>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}