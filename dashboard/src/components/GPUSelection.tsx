import { useState, useEffect } from 'react';
import { Search, ChevronLeft, ChevronRight, RefreshCw } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { apiClient } from '@/api/client';

interface GPUSelectionProps {
  selectedGpu: string;
  onGpuSelect: (gpu: string) => void;
  telemetryDataPoints?: number;
}

const ITEMS_PER_PAGE = 12; // Good number for GPU grid display

export function GPUSelection({ selectedGpu, onGpuSelect, telemetryDataPoints = 0 }: GPUSelectionProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [gpus, setGpus] = useState<string[]>([]);
  const [hosts, setHosts] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load GPU and host data
  const loadData = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const [gpuData, hostData] = await Promise.all([
        apiClient.getGpus(),
        apiClient.getHosts(),
      ]);
      setGpus(gpuData || []);
      setHosts(hostData || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setIsLoading(false);
    }
  };

  // Load data on component mount
  useEffect(() => {
    loadData();
  }, []);

  // Filter GPUs based on search term
  const filteredGpus = gpus.filter(gpu => 
    gpu.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Calculate pagination
  const totalPages = Math.ceil(filteredGpus.length / ITEMS_PER_PAGE);
  const startIndex = (currentPage - 1) * ITEMS_PER_PAGE;
  const endIndex = startIndex + ITEMS_PER_PAGE;
  const paginatedGpus = filteredGpus.slice(startIndex, endIndex);

  // Reset page when search changes
  const handleSearchChange = (value: string) => {
    setSearchTerm(value);
    setCurrentPage(1);
  };

  const handleGpuSelect = (gpu: string) => {
    onGpuSelect(gpu);
  };

  const handlePrevPage = () => {
    setCurrentPage(prev => Math.max(prev - 1, 1));
  };

  const handleNextPage = () => {
    setCurrentPage(prev => Math.min(prev + 1, totalPages));
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>GPU Selection</CardTitle>
            <CardDescription>
              {gpus.length} GPUs available across {hosts.length} hosts
            </CardDescription>
          </div>
          <button
            onClick={loadData}
            disabled={isLoading}
            className="flex items-center px-3 py-2 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-3">
            <p className="text-red-800 text-sm">Error: {error}</p>
          </div>
        )}

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 h-4 w-4" />
          <input
            type="text"
            placeholder="Search GPUs..."
            value={searchTerm}
            onChange={(e) => handleSearchChange(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* GPU Grid */}
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2">
          {paginatedGpus.map((gpu) => (
            <button
              key={gpu}
              onClick={() => handleGpuSelect(gpu)}
              className={`px-3 py-2 rounded text-sm border transition-all duration-200 hover:shadow-md ${
                selectedGpu === gpu
                  ? 'bg-blue-600 text-white border-blue-600 shadow-md'
                  : 'bg-white hover:bg-gray-50 hover:text-gray-900 border-gray-200'
              }`}
            >
              {gpu}
            </button>
          ))}
        </div>

        {/* Empty state */}
        {filteredGpus.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            {searchTerm ? `No GPUs found matching "${searchTerm}"` : 'No GPUs available'}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between">
            <div className="text-sm text-gray-600">
              Showing {startIndex + 1}-{Math.min(endIndex, filteredGpus.length)} of {filteredGpus.length} GPUs
            </div>
            <div className="flex items-center space-x-2">
              <button
                onClick={handlePrevPage}
                disabled={currentPage === 1}
                className="flex items-center px-3 py-1 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronLeft className="h-4 w-4 mr-1" />
                Previous
              </button>
              <span className="px-3 py-1 text-sm">
                Page {currentPage} of {totalPages}
              </span>
              <button
                onClick={handleNextPage}
                disabled={currentPage === totalPages}
                className="flex items-center px-3 py-1 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next
                <ChevronRight className="h-4 w-4 ml-1" />
              </button>
            </div>
          </div>
        )}

        {/* Selected GPU Info */}
        {selectedGpu && (
          <div className="pt-2 border-t border-gray-200">
            <p className="text-sm text-gray-600">
              Selected: <span className="font-medium text-gray-900">{selectedGpu}</span> • {telemetryDataPoints} data points
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}