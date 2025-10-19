import { useEffect, useState } from 'react';
import { Search, Server, Cpu, ChevronRight, ChevronLeft } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { apiClient } from '@/api/client';
import { POLLING_INTERVALS } from '@/lib/config';

interface HostInfo {
  hostname: string;
  gpuCount: number;
  gpus: string[];
}

const HOSTS_PER_PAGE = 5;

export function HostsOverview() {
  const [hosts, setHosts] = useState<HostInfo[]>([]);
  const [filteredHosts, setFilteredHosts] = useState<HostInfo[]>([]);
  const [totalHosts, setTotalHosts] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedHost, setExpandedHost] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);

  // Fetch hosts data
  const fetchHostsData = async () => {
    try {
      setError(null);
      
      // Get list of hosts
      const hostsResponse = await apiClient.getHostsWithMetadata();
      const hostList = hostsResponse.hosts || [];
      setTotalHosts(hostsResponse.total || 0);

      // Get GPU count for each host
      const hostsWithGpus = await Promise.all(
        hostList.map(async (hostname: string) => {
          try {
            const hostGpus = await apiClient.getHostGpus(hostname);
            return {
              hostname,
              gpuCount: hostGpus.length,
              gpus: hostGpus,
            };
          } catch (error) {
            console.error(`Failed to fetch GPUs for host ${hostname}:`, error);
            return {
              hostname,
              gpuCount: 0,
              gpus: [],
            };
          }
        })
      );

      setHosts(hostsWithGpus);
      setFilteredHosts(hostsWithGpus);
      setIsLoading(false);
    } catch (error) {
      console.error('Failed to fetch hosts data:', error);
      setError(error instanceof Error ? error.message : 'Failed to fetch hosts');
      setIsLoading(false);
    }
  };

  // Filter hosts based on search query
  useEffect(() => {
    if (!searchQuery.trim()) {
      setFilteredHosts(hosts);
    } else {
      const filtered = hosts.filter(host =>
        host.hostname.toLowerCase().includes(searchQuery.toLowerCase())
      );
      setFilteredHosts(filtered);
    }
    // Reset to first page when search changes
    setCurrentPage(1);
  }, [hosts, searchQuery]);

  // Initial load and polling
  useEffect(() => {
    fetchHostsData();
    const interval = setInterval(fetchHostsData, POLLING_INTERVALS.DASHBOARD);
    return () => clearInterval(interval);
  }, []);

  const toggleHostExpansion = (hostname: string) => {
    setExpandedHost(expandedHost === hostname ? null : hostname);
  };

  // Calculate pagination
  const totalPages = Math.ceil(filteredHosts.length / HOSTS_PER_PAGE);
  const startIndex = (currentPage - 1) * HOSTS_PER_PAGE;
  const endIndex = startIndex + HOSTS_PER_PAGE;
  const paginatedHosts = filteredHosts.slice(startIndex, endIndex);

  const handlePrevPage = () => {
    setCurrentPage(prev => Math.max(prev - 1, 1));
  };

  const handleNextPage = () => {
    setCurrentPage(prev => Math.min(prev + 1, totalPages));
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Hosts Overview
          </CardTitle>
          <CardDescription>Loading hosts information...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5" />
            Hosts Overview
          </CardTitle>
          <CardDescription>Error loading hosts data</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <p className="text-red-800 font-medium">Failed to load hosts</p>
            <p className="text-red-600 text-sm mt-1">{error}</p>
            <button
              onClick={fetchHostsData}
              className="mt-3 px-3 py-1 bg-red-600 text-white rounded text-sm hover:bg-red-700"
            >
              Retry
            </button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Hosts Overview
        </CardTitle>
        <CardDescription>
          {totalHosts} total hosts • {filteredHosts.length} shown
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Search Bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-gray-400" />
          <input
            type="text"
            placeholder="Search hosts by hostname..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
            >
              ✕
            </button>
          )}
        </div>

        {/* Summary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-blue-600" />
              <div>
                <p className="text-sm text-gray-600">Total Hosts</p>
                <p className="text-2xl font-bold text-blue-600">{totalHosts}</p>
              </div>
            </div>
          </div>
          <div className="bg-green-50 border border-green-200 rounded-lg p-4">
            <div className="flex items-center gap-2">
              <Cpu className="h-5 w-5 text-green-600" />
              <div>
                <p className="text-sm text-gray-600">Total GPUs</p>
                <p className="text-2xl font-bold text-green-600">
                  {hosts.reduce((sum, host) => sum + host.gpuCount, 0)}
                </p>
              </div>
            </div>
          </div>
          <div className="bg-purple-50 border border-purple-200 rounded-lg p-4">
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-purple-600" />
              <div>
                <p className="text-sm text-gray-600">Avg GPUs/Host</p>
                <p className="text-2xl font-bold text-purple-600">
                  {hosts.length > 0 ? (hosts.reduce((sum, host) => sum + host.gpuCount, 0) / hosts.length).toFixed(1) : '0'}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Hosts Table */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold">Hosts</h3>
            {totalPages > 1 && (
              <div className="text-sm text-gray-600">
                Showing {startIndex + 1}-{Math.min(endIndex, filteredHosts.length)} of {filteredHosts.length}
              </div>
            )}
          </div>
          
          {filteredHosts.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              {searchQuery ? 'No hosts match your search' : 'No hosts found'}
            </div>
          ) : (
            <div className="space-y-2">
              {paginatedHosts.map((host) => (
                <div key={host.hostname} className="border border-gray-200 rounded-lg">
                  <div
                    className="flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50"
                    onClick={() => toggleHostExpansion(host.hostname)}
                  >
                    <div className="flex items-center gap-3">
                      <Server className="h-4 w-4 text-gray-500" />
                      <div>
                        <p className="font-medium">{host.hostname}</p>
                        <p className="text-sm text-gray-600">
                          {host.gpuCount} GPU{host.gpuCount !== 1 ? 's' : ''}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={host.gpuCount > 0 ? 'default' : 'secondary'}>
                        {host.gpuCount} GPUs
                      </Badge>
                      <ChevronRight
                        className={`h-4 w-4 text-gray-400 transition-transform ${
                          expandedHost === host.hostname ? 'rotate-90' : ''
                        }`}
                      />
                    </div>
                  </div>
                  
                  {/* Expanded GPU List */}
                  {expandedHost === host.hostname && host.gpus.length > 0 && (
                    <div className="border-t border-gray-200 p-4 bg-gray-50">
                      <h4 className="text-sm font-medium text-gray-700 mb-2">GPUs on this host:</h4>
                      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                        {host.gpus.map((gpu) => (
                          <div
                            key={gpu}
                            className="bg-white border border-gray-200 rounded px-3 py-2 text-sm font-mono"
                          >
                            <Cpu className="inline h-3 w-3 mr-1 text-gray-500" />
                            {gpu}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Pagination Controls */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center space-x-2 pt-4">
              <button
                onClick={handlePrevPage}
                disabled={currentPage === 1}
                className="flex items-center px-3 py-2 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronLeft className="h-4 w-4 mr-1" />
                Previous
              </button>
              <span className="px-3 py-2 text-sm">
                Page {currentPage} of {totalPages}
              </span>
              <button
                onClick={handleNextPage}
                disabled={currentPage === totalPages}
                className="flex items-center px-3 py-2 border border-gray-200 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next
                <ChevronRight className="h-4 w-4 ml-1" />
              </button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}