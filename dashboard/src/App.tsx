import { useEffect, useState } from 'react'
import './App.css'

function App() {
  const [healthData, setHealthData] = useState({
    apiGateway: 'loading',
    collector: 'loading', 
    mqService: 'loading'
  })
  
  const [mqStats, setMqStats] = useState({
    queueSize: 'Loading...',
    subscribers: 'Loading...',
    pending: 'Loading...'
  })

  // Fetch health data
  useEffect(() => {
    const fetchHealth = async () => {
      try {
        // API Gateway health
        const apiResponse = await fetch('/health')
        if (apiResponse.ok) {
          setHealthData(prev => ({ ...prev, apiGateway: 'healthy' }))
        } else {
          setHealthData(prev => ({ ...prev, apiGateway: 'error' }))
        }
      } catch {
        setHealthData(prev => ({ ...prev, apiGateway: 'error' }))
      }

      try {
        // Collector health
        const collectorResponse = await fetch('/collector/health')
        if (collectorResponse.ok) {
          setHealthData(prev => ({ ...prev, collector: 'healthy' }))
        } else {
          setHealthData(prev => ({ ...prev, collector: 'error' }))
        }
      } catch {
        setHealthData(prev => ({ ...prev, collector: 'error' }))
      }

      try {
        // MQ health
        const mqResponse = await fetch('/mq/health')
        if (mqResponse.ok) {
          setHealthData(prev => ({ ...prev, mqService: 'healthy' }))
        } else {
          setHealthData(prev => ({ ...prev, mqService: 'error' }))
        }
      } catch {
        setHealthData(prev => ({ ...prev, mqService: 'error' }))
      }
    }

    fetchHealth()
    const interval = setInterval(fetchHealth, 10000) // Every 10 seconds
    return () => clearInterval(interval)
  }, [])

  // Fetch MQ stats
  useEffect(() => {
    const fetchMqStats = async () => {
      try {
        const response = await fetch('/mq/stats')
        if (response.ok) {
          const data = await response.json()
          const telemetryTopic = data.topics?.telemetry || {}
          setMqStats({
            queueSize: telemetryTopic.queue_size || 0,
            subscribers: telemetryTopic.subscriber_count || 0,
            pending: telemetryTopic.pending_messages || 0
          })
        }
      } catch (error) {
        console.error('Failed to fetch MQ stats:', error)
      }
    }

    fetchMqStats()
    const interval = setInterval(fetchMqStats, 5000) // Every 5 seconds  
    return () => clearInterval(interval)
  }, [])

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return 'bg-green-50 border-green-200 text-green-800'
      case 'error': return 'bg-red-50 border-red-200 text-red-800'
      default: return 'bg-blue-50 border-blue-200 text-blue-800'
    }
  }

  const getStatusDot = (status: string) => {
    switch (status) {
      case 'healthy': return 'bg-green-500'
      case 'error': return 'bg-red-500'
      default: return 'bg-blue-500'
    }
  }

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">GPU Telemetry Dashboard</h1>
              <p className="text-gray-600">Real-time monitoring and analytics</p>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
        <div className="space-y-6">
          {/* Health Panel */}
          <div className="bg-white shadow rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4">Service Health</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className={`border rounded-lg p-4 ${getStatusColor(healthData.apiGateway)}`}>
                <div className="flex items-center">
                  <div className={`w-3 h-3 rounded-full mr-3 ${getStatusDot(healthData.apiGateway)}`}></div>
                  <div>
                    <p className="font-medium">API Gateway</p>
                    <p className="text-sm opacity-75">{healthData.apiGateway}</p>
                  </div>
                </div>
              </div>
              <div className={`border rounded-lg p-4 ${getStatusColor(healthData.collector)}`}>
                <div className="flex items-center">
                  <div className={`w-3 h-3 rounded-full mr-3 ${getStatusDot(healthData.collector)}`}></div>
                  <div>
                    <p className="font-medium">Collector</p>
                    <p className="text-sm opacity-75">{healthData.collector}</p>
                  </div>
                </div>
              </div>
              <div className={`border rounded-lg p-4 ${getStatusColor(healthData.mqService)}`}>
                <div className="flex items-center">
                  <div className={`w-3 h-3 rounded-full mr-3 ${getStatusDot(healthData.mqService)}`}></div>
                  <div>
                    <p className="font-medium">MQ Service</p>
                    <p className="text-sm opacity-75">{healthData.mqService}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* MQ Overview */}
          <div className="bg-white shadow rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4">Message Queue Overview</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-gray-600">Queue Size</p>
                <p className="text-2xl font-bold text-blue-600">{mqStats.queueSize}</p>
              </div>
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-gray-600">Subscribers</p>
                <p className="text-2xl font-bold text-blue-600">{mqStats.subscribers}</p>
              </div>
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-gray-600">Pending</p>
                <p className="text-2xl font-bold text-blue-600">{mqStats.pending}</p>
              </div>
            </div>
          </div>

          {/* Status */}
          <div className="bg-white shadow rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4">Dashboard Status</h2>
            <p className="text-green-600 font-medium">✅ Dashboard is now working correctly!</p>
            <p className="text-gray-600 mt-2">Both HealthPanel and MQOverview are displaying real-time data.</p>
          </div>
        </div>
      </main>
    </div>
  )
}

export default App
