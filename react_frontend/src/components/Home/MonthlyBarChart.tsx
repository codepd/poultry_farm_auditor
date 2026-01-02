import React, { useState, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import './MonthlyBarChart.css';

interface MonthlyData {
  year: number;
  month: number;
  month_name: string;
  sales: number;
  feed_expense: number;
  medicine_expense: number;
  labor_expense: number;
  other_expense: number;
  total_expense: number;
  net_profit: number;
}

interface MonthlyBarChartProps {
  onMonthClick: (year: number, month: number) => void;
}

const MonthlyBarChart: React.FC<MonthlyBarChartProps> = ({ onMonthClick }) => {
  const { isAuthenticated } = useAuth();
  const [data, setData] = useState<MonthlyData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const response = await fetch(
          `${process.env.REACT_APP_API_URL || 'http://localhost:8080/api'}/analytics/last-12-months`,
          {
            headers: {
              'Authorization': `Bearer ${localStorage.getItem('token')}`,
            },
          }
        );
        const result = await response.json();
        if (result.success) {
          setData(result.data);
        } else {
          setError('Failed to load chart data');
        }
      } catch (err: any) {
        setError('Failed to load chart data');
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [isAuthenticated]);

  if (loading) {
    return <div className="chart-loading">Loading chart data...</div>;
  }

  if (error) {
    return <div className="chart-error">{error}</div>;
  }

  if (data.length === 0) {
    return <div className="chart-no-data">No data available for the last 12 months</div>;
  }

  // Format data for Recharts - show full month names and add click handlers
  const chartData = data.map((item) => {
    const date = new Date(item.year, item.month - 1);
    const monthFullName = date.toLocaleString('default', { month: 'long' });
    return {
      ...item,
      monthLabel: `${monthFullName.substring(0, 3)}\n${item.year}`,
      monthFullName,
      // Format for display
      salesDisplay: item.sales,
      expensesDisplay: item.total_expense,
      profitDisplay: item.net_profit,
    };
  });

  // Custom tooltip with better formatting
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="chart-tooltip">
          <div className="tooltip-header">
            {data.monthFullName} {data.year}
          </div>
          <div className="tooltip-item sales">
            <span className="tooltip-label">Sales (Income):</span>
            <span className="tooltip-value">₹{data.sales.toLocaleString('en-IN')}</span>
          </div>
          <div className="tooltip-item expense">
            <span className="tooltip-label">Total Expenses:</span>
            <span className="tooltip-value">₹{data.total_expense.toLocaleString('en-IN')}</span>
          </div>
          <div className="tooltip-breakdown">
            <div>• Feed: ₹{data.feed_expense.toLocaleString('en-IN')}</div>
            <div>• Medicine: ₹{data.medicine_expense.toLocaleString('en-IN')}</div>
            <div>• Labor: ₹{data.labor_expense.toLocaleString('en-IN')}</div>
            <div>• Other: ₹{data.other_expense.toLocaleString('en-IN')}</div>
          </div>
          <div className={`tooltip-item ${data.net_profit >= 0 ? 'profit' : 'loss'}`}>
            <span className="tooltip-label">Net Profit:</span>
            <span className="tooltip-value">₹{data.net_profit.toLocaleString('en-IN')}</span>
          </div>
          <div className="tooltip-note">Click to see full details</div>
        </div>
      );
    }
    return null;
  };

  // Format Y-axis values
  const formatYAxis = (value: number) => {
    if (value >= 100000) {
      return `₹${(value / 100000).toFixed(1)}L`;
    } else if (value >= 1000) {
      return `₹${(value / 1000).toFixed(0)}K`;
    }
    return `₹${value}`;
  };

  return (
    <div className="monthly-bar-chart">
      <div className="chart-header">
        <h3>Last 12 Months - Income & Expenses</h3>
        <div className="chart-subtitle">Click on any month to see detailed information</div>
      </div>
      <div className="chart-wrapper">
        <ResponsiveContainer width="100%" height={400}>
          <BarChart
            data={chartData}
            margin={{ top: 20, right: 30, left: 20, bottom: 60 }}
            onClick={(data: any) => {
              if (data && data.activePayload && data.activePayload[0]) {
                const item = data.activePayload[0].payload;
                onMonthClick(item.year, item.month);
              }
            }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
            <XAxis
              dataKey="monthLabel"
              angle={-45}
              textAnchor="end"
              height={80}
              tick={{ fontSize: 12, fill: '#666' }}
              interval={0}
            />
            <YAxis
              tick={{ fontSize: 12, fill: '#666' }}
              tickFormatter={formatYAxis}
              label={{ value: 'Amount (₹)', angle: -90, position: 'insideLeft', style: { fill: '#666' } }}
            />
            <Tooltip content={<CustomTooltip />} />
            <Legend
              wrapperStyle={{ paddingTop: '20px' }}
              iconType="square"
            />
            <Bar
              dataKey="salesDisplay"
              name="Sales (Income)"
              fill="#4CAF50"
              radius={[4, 4, 0, 0]}
              onClick={(data: any) => {
                if (data && data.payload) {
                  onMonthClick(data.payload.year, data.payload.month);
                }
              }}
            >
              {chartData.map((entry, index) => (
                <Cell
                  key={`cell-sales-${index}`}
                  cursor="pointer"
                  style={{ cursor: 'pointer' }}
                />
              ))}
            </Bar>
            <Bar
              dataKey="expensesDisplay"
              name="Total Expenses"
              fill="#f44336"
              radius={[4, 4, 0, 0]}
              onClick={(data: any) => {
                if (data && data.payload) {
                  onMonthClick(data.payload.year, data.payload.month);
                }
              }}
            >
              {chartData.map((entry, index) => (
                <Cell
                  key={`cell-expense-${index}`}
                  cursor="pointer"
                  style={{ cursor: 'pointer' }}
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div className="chart-summary">
        <div className="summary-item">
          <span className="summary-label">Total Sales (12 months):</span>
          <span className="summary-value sales">
            ₹{data.reduce((sum, d) => sum + d.sales, 0).toLocaleString('en-IN')}
          </span>
        </div>
        <div className="summary-item">
          <span className="summary-label">Total Expenses (12 months):</span>
          <span className="summary-value expense">
            ₹{data.reduce((sum, d) => sum + d.total_expense, 0).toLocaleString('en-IN')}
          </span>
        </div>
        <div className="summary-item">
          <span className="summary-label">Net Profit (12 months):</span>
          <span className={`summary-value ${data.reduce((sum, d) => sum + d.net_profit, 0) >= 0 ? 'profit' : 'loss'}`}>
            ₹{data.reduce((sum, d) => sum + d.net_profit, 0).toLocaleString('en-IN')}
          </span>
        </div>
      </div>
    </div>
  );
};

export default MonthlyBarChart;
