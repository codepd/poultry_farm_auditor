import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { henBatchesAPI, MortalityRecord } from '../../services/api';
import { useTenant } from '../../hooks/useTenant';
import { formatDateForTenant } from '../../utils/dateUtils';
import './MortalityChart.css';

interface MortalityChartProps {
  batchId: number;
  batchName: string;
  onClose: () => void;
}

const MortalityChart: React.FC<MortalityChartProps> = ({ batchId, batchName, onClose }) => {
  const { timezone, dateFormat } = useTenant();
  const [mortalityData, setMortalityData] = useState<MortalityRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    const fetchMortalityHistory = async () => {
      try {
        setLoading(true);
        setError('');
        const data = await henBatchesAPI.getMortalityHistory(batchId);
        setMortalityData(data);
      } catch (err: any) {
        console.error('Error fetching mortality history:', err);
        console.error('Error details:', err.response?.data);
        const errorMessage = err.response?.data?.error || err.message || 'Failed to load mortality history';
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchMortalityHistory();
  }, [batchId]);

  // Group mortality by date and calculate cumulative mortality
  const processChartData = () => {
    if (mortalityData.length === 0) return [];

    // Group by date
    const dateMap = new Map<string, number>();
    mortalityData.forEach(record => {
      const date = record.mortality_date;
      const current = dateMap.get(date) || 0;
      dateMap.set(date, current + record.count);
    });

    // Convert to array and sort by date
    const sortedDates = Array.from(dateMap.entries()).sort((a, b) => 
      new Date(a[0]).getTime() - new Date(b[0]).getTime()
    );

    // Calculate cumulative mortality
    let cumulative = 0;
    return sortedDates.map(([date, count]) => {
      cumulative += count;
      return {
        date,
        daily: count,
        cumulative,
        formattedDate: formatDateForTenant(date, timezone, dateFormat),
      };
    });
  };

  const chartData = processChartData();
  const totalMortality = mortalityData.reduce((sum, record) => sum + record.count, 0);

  return (
    <div className="mortality-chart-modal-overlay" onClick={onClose}>
      <div className="mortality-chart-modal" onClick={(e) => e.stopPropagation()}>
        <div className="mortality-chart-header">
          <h2>Mortality History - {batchName}</h2>
          <button className="close-btn" onClick={onClose}>×</button>
        </div>

        <div className="mortality-chart-content">
          {loading ? (
            <div className="loading">Loading mortality history...</div>
          ) : error ? (
            <div className="error">{error}</div>
          ) : chartData.length === 0 ? (
            <div className="no-data">No mortality records found for this batch.</div>
          ) : (
            <>
              <div className="mortality-summary">
                <div className="summary-item">
                  <span className="summary-label">Total Mortality:</span>
                  <span className="summary-value">{totalMortality.toLocaleString()}</span>
                </div>
                <div className="summary-item">
                  <span className="summary-label">Records:</span>
                  <span className="summary-value">{mortalityData.length}</span>
                </div>
              </div>

              <div className="chart-container">
                <h3>Daily Mortality</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis 
                      dataKey="formattedDate" 
                      angle={-45}
                      textAnchor="end"
                      height={80}
                      interval={Math.floor(chartData.length / 10)}
                    />
                    <YAxis />
                    <Tooltip 
                      formatter={(value: number) => [value.toLocaleString(), 'Count']}
                      labelFormatter={(label) => `Date: ${label}`}
                    />
                    <Legend />
                    <Line 
                      type="monotone" 
                      dataKey="daily" 
                      stroke="#d32f2f" 
                      strokeWidth={2}
                      name="Daily Mortality"
                      dot={{ r: 4 }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>

              <div className="chart-container">
                <h3>Cumulative Mortality</h3>
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis 
                      dataKey="formattedDate" 
                      angle={-45}
                      textAnchor="end"
                      height={80}
                      interval={Math.floor(chartData.length / 10)}
                    />
                    <YAxis />
                    <Tooltip 
                      formatter={(value: number) => [value.toLocaleString(), 'Cumulative']}
                      labelFormatter={(label) => `Date: ${label}`}
                    />
                    <Legend />
                    <Line 
                      type="monotone" 
                      dataKey="cumulative" 
                      stroke="#667eea" 
                      strokeWidth={2}
                      name="Cumulative Mortality"
                      dot={{ r: 4 }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>

              <div className="mortality-table">
                <h3>Mortality Records</h3>
                <table>
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Count</th>
                      <th>Reason</th>
                      <th>Notes</th>
                    </tr>
                  </thead>
                  <tbody>
                    {mortalityData
                      .sort((a, b) => new Date(b.mortality_date).getTime() - new Date(a.mortality_date).getTime())
                      .map((record) => (
                        <tr key={record.id}>
                          <td>{formatDateForTenant(record.mortality_date, timezone, dateFormat)}</td>
                          <td>{record.count.toLocaleString()}</td>
                          <td>{record.reason || '-'}</td>
                          <td>{record.notes || '-'}</td>
                        </tr>
                      ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default MortalityChart;

