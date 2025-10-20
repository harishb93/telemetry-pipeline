import { useState } from 'react';
import { Search, Loader2, Database } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { apiClient } from '@/api/client';
import type { TelemetryResponse } from '@/lib/types';

interface GPUSelectionProps {
  selectedGpu: string;
  onGpuSelect: (gpu: string) => void;
  telemetryDataPoints?: number;
}

export function GPUSelection({ selectedGpu, onGpuSelect }: GPUSelectionProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [telemetryData, setTelemetryData] = useState<TelemetryResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async () => {
    if (!searchTerm.trim()) {
      setError('Please enter a GPU ID to search');
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      setTelemetryData(null);
      
      console.log('GPUSelection: Searching for GPU:', searchTerm);
      const response = await apiClient.getTelemetry(searchTerm.trim(), { limit: 100 });
      
      if (response && response.data && response.data.length > 0) {
        setTelemetryData(response);
        onGpuSelect(searchTerm.trim());
      } else {
        setError(`No telemetry data found for GPU: ${searchTerm}`);
        setTelemetryData(null);
      }
    } catch (err) {
      console.error('GPUSelection: Error fetching telemetry:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch telemetry data');
      setTelemetryData(null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  const formatJSON = (data: any): string => {
    return JSON.stringify(data, null, 2);
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Database className="h-5 w-5" />
          GPU Telemetry Search
        </CardTitle>
        <CardDescription>
          Enter a GPU ID to search for telemetry data
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Search Input */}
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
            <input
              type="text"
              placeholder="Enter GPU ID (e.g., GPU-11111111-2222-3333-4444-555555555555)"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              onKeyPress={handleKeyPress}
              className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              disabled={isLoading}
            />
          </div>
          <button
            onClick={handleSearch}
            disabled={isLoading || !searchTerm.trim()}
            className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
            ) : (
              <Search className="h-4 w-4 mr-2" />
            )}
            Search
          </button>
        </div>

        {/* Error Message */}
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-3">
            <p className="text-red-800 text-sm">{error}</p>
          </div>
        )}

        {/* Loading State */}
        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <div className="text-center">
              <Loader2 className="h-6 w-6 animate-spin mx-auto text-blue-600" />
              <p className="mt-2 text-sm text-gray-600">Searching for telemetry data...</p>
            </div>
          </div>
        )}

        {/* Telemetry Data Display */}
        {telemetryData && !isLoading && (
          <div className="space-y-4">
            {/* Summary */}
            <div className="bg-green-50 border border-green-200 rounded-lg p-3">
              <p className="text-green-800 text-sm font-medium">
                ✅ Found {telemetryData.total} telemetry entries for GPU: {selectedGpu}
              </p>
              <p className="text-green-700 text-xs mt-1">
                Showing {telemetryData.data?.length || 0} entries (limit: {telemetryData.pagination.limit})
              </p>
            </div>

            {/* JSON Response Display */}
            <div className="space-y-2">
              <h4 className="text-sm font-medium text-gray-900">Telemetry Data Response:</h4>
              <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 max-h-96 overflow-auto">
                <pre className="text-xs text-gray-800 whitespace-pre-wrap font-mono">
                  {formatJSON(telemetryData)}
                </pre>
              </div>
            </div>

            {/* Quick Stats */}
            {telemetryData.data && telemetryData.data.length > 0 && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-2 border-t border-gray-200">
                <div className="text-center p-2 bg-blue-50 rounded">
                  <p className="text-xs text-gray-600">Total Entries</p>
                  <p className="text-lg font-semibold text-blue-600">{telemetryData.total}</p>
                </div>
                <div className="text-center p-2 bg-green-50 rounded">
                  <p className="text-xs text-gray-600">Returned</p>
                  <p className="text-lg font-semibold text-green-600">{telemetryData.data.length}</p>
                </div>
                <div className="text-center p-2 bg-purple-50 rounded">
                  <p className="text-xs text-gray-600">GPU ID</p>
                  <p className="text-xs font-medium text-purple-600 truncate" title={selectedGpu}>
                    {selectedGpu.split('-')[0]}...
                  </p>
                </div>
                <div className="text-center p-2 bg-orange-50 rounded">
                  <p className="text-xs text-gray-600">Host</p>
                  <p className="text-xs font-medium text-orange-600">
                    {telemetryData.data[0]?.hostname || 'N/A'}
                  </p>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Instructions */}
        {!telemetryData && !isLoading && !error && (
          <div className="text-center py-8 text-gray-500">
            <Database className="h-8 w-8 mx-auto mb-2 text-gray-400" />
            <p className="text-sm">Enter a GPU ID above to search for telemetry data</p>
            <p className="text-xs mt-1 text-gray-400">Press Enter or click Search to fetch data</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}