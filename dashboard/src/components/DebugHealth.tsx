import { useEffect, useState } from 'react';

export function DebugHealth() {
  const [collectorStatus, setCollectorStatus] = useState('loading');
  const [mqStatus, setMqStatus] = useState('loading');
  const [apiStatus, setApiStatus] = useState('loading');
  const [errors, setErrors] = useState<string[]>([]);

  useEffect(() => {
    const testHealthEndpoints = async () => {
      const newErrors: string[] = [];

      // Test Collector (via proxy)
      try {
        const collectorResponse = await fetch('/collector/health');
        const collectorData = await collectorResponse.json();
        console.log('Collector response:', collectorData);
        setCollectorStatus(collectorData.status || 'unknown');
      } catch (error) {
        console.error('Collector error:', error);
        newErrors.push(`Collector: ${error}`);
        setCollectorStatus('error');
      }

      // Test MQ (via proxy)
      try {
        const mqResponse = await fetch('/mq/health');
        const mqData = await mqResponse.json();
        console.log('MQ response:', mqData);
        setMqStatus(mqData.status || 'unknown');
      } catch (error) {
        console.error('MQ error:', error);
        newErrors.push(`MQ: ${error}`);
        setMqStatus('error');
      }

      // Test API Gateway (via proxy)
      try {
        const apiResponse = await fetch('/api/health');
        const apiData = await apiResponse.json();
        console.log('API Gateway response:', apiData);
        setApiStatus(apiData.status || 'unknown');
      } catch (error) {
        console.error('API Gateway error:', error);
        newErrors.push(`API Gateway: ${error}`);
        setApiStatus('error');
      }

      setErrors(newErrors);
    };

    testHealthEndpoints();
  }, []);

  return (
    <div className="p-4 border rounded">
      <h3 className="font-bold mb-2">Debug Health Status</h3>
      <div className="space-y-2">
        <div>Collector: <span className={collectorStatus === 'healthy' ? 'text-green-600' : 'text-red-600'}>{collectorStatus}</span></div>
        <div>MQ Service: <span className={mqStatus === 'healthy' ? 'text-green-600' : 'text-red-600'}>{mqStatus}</span></div>
        <div>API Gateway: <span className={apiStatus === 'healthy' ? 'text-green-600' : 'text-red-600'}>{apiStatus}</span></div>
      </div>
      {errors.length > 0 && (
        <div className="mt-4">
          <h4 className="font-semibold text-red-600">Errors:</h4>
          <ul className="text-sm text-red-600">
            {errors.map((error, i) => (
              <li key={i}>{error}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}