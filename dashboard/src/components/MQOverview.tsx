import { useState, useEffect } from 'react';
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from 'recharts';
import { RefreshCw, AlertCircle, Database, Users, MessageSquare, Clock } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/api/client';
import { usePolling } from '@/lib/usePolling';
import { POLLING_INTERVALS } from '@/lib/config';
import type { MQStats, TopicStats } from '@/lib/types';

interface TopicWithName extends TopicStats {
  name: string;
}

interface MQOverviewProps {
  className?: string;
}

function TopicMetricsCard({ topic }: { topic: TopicWithName }) {
  const utilizationPercentage = topic.queue_size > 0 
    ? Math.round((topic.pending_messages / topic.queue_size) * 100)
    : 0;

  const pieData = [
    { name: 'Processed', value: topic.queue_size - topic.pending_messages, color: '#82ca9d' },
    { name: 'Pending', value: topic.pending_messages, color: '#ff7300' }
  ];

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium flex items-center space-x-2">
          <Database className="h-4 w-4" />
          <span>{topic.name}</span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-3 gap-2 mb-4">
          <div className="text-center">
            <div className="text-lg font-semibold text-blue-600">{topic.queue_size}</div>
            <div className="text-xs text-gray-500">Queue Size</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-semibold text-orange-600">{topic.pending_messages}</div>
            <div className="text-xs text-gray-500">Pending</div>
          </div>
          <div className="text-center">
            <div className="text-lg font-semibold text-green-600">{topic.subscriber_count}</div>
            <div className="text-xs text-gray-500">Subscribers</div>
          </div>
        </div>

        {/* Mini Pie Chart */}
        {topic.queue_size > 0 && (
          <div className="h-24">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  innerRadius={15}
                  outerRadius={30}
                  paddingAngle={2}
                  dataKey="value"
                >
                  {pieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip 
                  formatter={(value, name) => [`${value} messages`, name]}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}

        <div className="mt-2">
          <Badge variant={utilizationPercentage > 80 ? 'destructive' : utilizationPercentage > 50 ? 'secondary' : 'default'}>
            {utilizationPercentage}% Utilization
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}

function TopicsTable({ topics }: { topics: TopicWithName[] }) {
  if (topics.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Topics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-center text-gray-500 py-8">
            <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>No topics found</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">Topics Details</CardTitle>
        <CardDescription>{topics.length} topics active</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200">
                <th className="text-left py-2 font-medium">Topic</th>
                <th className="text-center py-2 font-medium">
                  <div className="flex items-center justify-center space-x-1">
                    <Database className="h-3 w-3" />
                    <span>Queue</span>
                  </div>
                </th>
                <th className="text-center py-2 font-medium">
                  <div className="flex items-center justify-center space-x-1">
                    <MessageSquare className="h-3 w-3" />
                    <span>Pending</span>
                  </div>
                </th>
                <th className="text-center py-2 font-medium">
                  <div className="flex items-center justify-center space-x-1">
                    <Users className="h-3 w-3" />
                    <span>Subscribers</span>
                  </div>
                </th>
                <th className="text-center py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {topics.map((topic) => {
                const utilizationPercentage = topic.queue_size > 0 
                  ? Math.round((topic.pending_messages / topic.queue_size) * 100)
                  : 0;
                
                return (
                  <tr key={topic.name} className="border-b border-gray-100 hover:bg-gray-50">
                    <td className="py-3 font-medium">{topic.name}</td>
                    <td className="text-center py-3">
                      <Badge variant="outline">{topic.queue_size}</Badge>
                    </td>
                    <td className="text-center py-3">
                      <Badge variant={topic.pending_messages > 0 ? 'secondary' : 'default'}>
                        {topic.pending_messages}
                      </Badge>
                    </td>
                    <td className="text-center py-3">
                      <Badge variant={topic.subscriber_count > 0 ? 'default' : 'destructive'}>
                        {topic.subscriber_count}
                      </Badge>
                    </td>
                    <td className="text-center py-3">
                      <Badge 
                        variant={
                          utilizationPercentage > 80 ? 'destructive' : 
                          utilizationPercentage > 50 ? 'secondary' : 
                          'default'
                        }
                      >
                        {utilizationPercentage}%
                      </Badge>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

export function MQOverview({ className }: MQOverviewProps) {
  const [isPolling, setIsPolling] = useState(true);
  const [lastUpdate, setLastUpdate] = useState<string>('');
  
  const { data: mqStats, isLoading, error, refetch } = usePolling<MQStats>(
    () => apiClient.getMQStats(),
    { 
      interval: POLLING_INTERVALS.STATS,
      enabled: isPolling 
    }
  );

  // Transform stats data for easier handling
  const topics: TopicWithName[] = mqStats ? 
    Object.entries(mqStats.topics).map(([name, stats]) => ({
      name,
      ...stats
    })) : [];

  // Update last update timestamp when data changes
  useEffect(() => {
    if (mqStats && !isLoading) {
      setLastUpdate(new Date().toISOString());
    }
  }, [mqStats, isLoading]);

  // Calculate summary stats
  const totalMessages = topics.reduce((sum, topic) => sum + topic.queue_size, 0);
  const totalPending = topics.reduce((sum, topic) => sum + topic.pending_messages, 0);
  const totalSubscribers = topics.reduce((sum, topic) => sum + topic.subscriber_count, 0);
  const overallUtilization = totalMessages > 0 ? Math.round((totalPending / totalMessages) * 100) : 0;

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">Message Queue Overview</h2>
          <p className="text-sm text-gray-600">
            {topics.length} topics • {totalMessages} total messages • {totalPending} pending
            {lastUpdate && (
              <span className="ml-2">
                • Updated {new Date(lastUpdate).toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
        
        <div className="flex items-center space-x-2">
          <Badge 
            variant={
              error ? 'destructive' : 
              overallUtilization > 80 ? 'destructive' :
              overallUtilization > 50 ? 'secondary' : 
              'default'
            }
          >
            {error ? 'Error' : 
             isLoading ? 'Loading...' :
             `${overallUtilization}% Utilization`}
          </Badge>
          
          <button
            onClick={() => setIsPolling(!isPolling)}
            className="text-xs px-2 py-1 rounded border hover:bg-gray-50 transition-colors"
          >
            {isPolling ? 'Pause' : 'Resume'}
          </button>
          
          <button
            onClick={refetch}
            disabled={isLoading}
            className="text-xs px-2 py-1 rounded border hover:bg-gray-50 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-3 w-3 ${isLoading ? 'animate-spin' : ''}`} />
          </button>
        </div>
      </div>

      {/* Error State */}
      {error && (
        <Card className="border-red-200 bg-red-50 mb-4">
          <CardContent className="pt-4">
            <div className="flex items-center space-x-2 text-red-600">
              <AlertCircle className="h-4 w-4" />
              <span className="text-sm">Failed to fetch MQ stats: {error.message}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Summary Cards */}
      {!error && (
        <div className="grid gap-4 md:grid-cols-4 mb-6">
          <Card>
            <CardContent className="pt-4">
              <div className="flex items-center space-x-2">
                <Database className="h-4 w-4 text-blue-500" />
                <div>
                  <div className="text-2xl font-bold">{topics.length}</div>
                  <div className="text-xs text-gray-500">Topics</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="pt-4">
              <div className="flex items-center space-x-2">
                <MessageSquare className="h-4 w-4 text-green-500" />
                <div>
                  <div className="text-2xl font-bold">{totalMessages}</div>
                  <div className="text-xs text-gray-500">Total Messages</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="pt-4">
              <div className="flex items-center space-x-2">
                <Clock className="h-4 w-4 text-orange-500" />
                <div>
                  <div className="text-2xl font-bold">{totalPending}</div>
                  <div className="text-xs text-gray-500">Pending</div>
                </div>
              </div>
            </CardContent>
          </Card>
          
          <Card>
            <CardContent className="pt-4">
              <div className="flex items-center space-x-2">
                <Users className="h-4 w-4 text-purple-500" />
                <div>
                  <div className="text-2xl font-bold">{totalSubscribers}</div>
                  <div className="text-xs text-gray-500">Subscribers</div>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      <div className="space-y-6">
        {/* Individual Topic Cards */}
        {topics.length > 0 && (
          <div>
            <h3 className="text-md font-medium mb-3">Topic Details</h3>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 mb-6">
              {topics.map((topic) => (
                <TopicMetricsCard key={topic.name} topic={topic} />
              ))}
            </div>
          </div>
        )}
        
        {/* Topics Table */}
        <TopicsTable topics={topics} />
      </div>
    </div>
  );
}