import React, { useState, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';
import { getOrCreateDeviceId } from '../../utils/deviceId';
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
  layer_feed_expense?: number;
  rearing_feed_expense?: number;
  medicine_expense: number;
  labor_expense: number;
  other_expense: number;
  other_income: number;
  total_discounts: number;
  total_tds: number;
  total_expense: number;
  flock_operating_profit?: number;
  ledger_net_profit?: number;
  farm_net_profit?: number;
  net_profit: number;
}

interface MonthlyBarChartProps {
  onMonthClick: (year: number, month: number) => void;
  disableTooltipOnMobile?: boolean;
}

const MonthlyBarChart: React.FC<MonthlyBarChartProps> = ({
  onMonthClick,
  disableTooltipOnMobile = false,
}) => {
  const { isAuthenticated } = useAuth();
  const { t, language } = useLanguage();
  const [data, setData] = useState<MonthlyData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [isMobile, setIsMobile] = useState(() => window.innerWidth <= 768);

  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth <= 768);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;

    const fetchData = async () => {
      setLoading(true);
      setError('');
      try {
        const response = await fetch(
          `${process.env.REACT_APP_API_URL || 'http://localhost:8080/api'}/analytics/last-12-months`,
          {
            credentials: 'include',
            headers: {
              Authorization: `Bearer ${localStorage.getItem('token')}`,
              'X-Device-Id': getOrCreateDeviceId(),
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
    const locale = language === 'ta' ? 'ta-IN' : 'en-IN';
    const localizedShortMonth = new Date(Date.UTC(item.year, item.month - 1, 1))
      .toLocaleString(locale, { month: 'short' });
    const localizedFullMonth = new Date(Date.UTC(item.year, item.month - 1, 1))
      .toLocaleString(locale, { month: 'long' });
    return {
      ...item,
      monthLabel: `${localizedShortMonth}\n${item.year}`,
      monthFullName: localizedFullMonth,
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
            <span className="tooltip-label">Sales (Egg Income):</span>
            <span className="tooltip-value">₹{data.sales.toLocaleString('en-IN')}</span>
          </div>
          <div className="tooltip-item expense">
            <span className="tooltip-label">Total Expenses:</span>
            <span className="tooltip-value">₹{data.total_expense.toLocaleString('en-IN')}</span>
          </div>
          <div className="tooltip-breakdown">
            <div>• Total Feed: ₹{data.feed_expense.toLocaleString('en-IN')}</div>
            {(data.rearing_feed_expense || 0) > 0 && (
              <>
                <div style={{ paddingLeft: '10px', fontSize: '0.82em', opacity: 0.9 }}>
                  ↳ Layer Feed: ₹{(data.layer_feed_expense || 0).toLocaleString('en-IN')}
                </div>
                <div style={{ paddingLeft: '10px', fontSize: '0.82em', opacity: 0.9 }}>
                  ↳ Chick/Grower Feed: ₹{(data.rearing_feed_expense || 0).toLocaleString('en-IN')}
                </div>
              </>
            )}
            <div>• Medicine: ₹{data.medicine_expense.toLocaleString('en-IN')}</div>
            <div>• Labor: ₹{data.labor_expense.toLocaleString('en-IN')}</div>
            <div>• Other Overheads: ₹{data.other_expense.toLocaleString('en-IN')}</div>
            <div>• Other Income (Hen Sales): ₹{(data.other_income || 0).toLocaleString('en-IN')}</div>
            <div>• TDS: ₹{(data.total_tds || 0).toLocaleString('en-IN')}</div>
            <div>• Discounts: ₹{(data.total_discounts || 0).toLocaleString('en-IN')}</div>
          </div>
          {data.ledger_net_profit !== undefined && (
            <div className="tooltip-item" style={{ fontSize: '0.85em', borderTop: '1px dashed #e2e8f0', paddingTop: '4px' }}>
              <span className="tooltip-label">Statement Net Profit:</span>
              <span className="tooltip-value">₹{data.ledger_net_profit.toLocaleString('en-IN')}</span>
            </div>
          )}
          <div className={`tooltip-item ${data.net_profit >= 0 ? 'profit' : 'loss'}`}>
            <span className="tooltip-label">Farm Overall Profit:</span>
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

  const shouldShowTooltip = !(disableTooltipOnMobile && isMobile);

  return (
    <div className="monthly-bar-chart">
      <div className="chart-header">
        <h3>{t('chart.title12Months')}</h3>
      </div>
      <div className="chart-wrapper">
        <ResponsiveContainer width="100%" height={isMobile ? 280 : 400}>
          <BarChart
            data={chartData}
            margin={isMobile ? { top: 8, right: 8, left: 0, bottom: 44 } : { top: 20, right: 30, left: 20, bottom: 60 }}
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
              angle={isMobile ? -35 : -45}
              textAnchor="end"
              height={isMobile ? 56 : 80}
              tick={{ fontSize: isMobile ? 10 : 12, fill: '#666' }}
              interval={0}
            />
            <YAxis
              tick={{ fontSize: isMobile ? 10 : 12, fill: '#666' }}
              tickFormatter={formatYAxis}
              label={isMobile ? undefined : { value: '₹', angle: -90, position: 'insideLeft', style: { fill: '#666' } }}
            />
            {shouldShowTooltip && <Tooltip content={<CustomTooltip />} />}
            <Legend
              wrapperStyle={{ paddingTop: isMobile ? '6px' : '20px', fontSize: isMobile ? '11px' : '13px' }}
              iconType="square"
            />
            <Bar
              dataKey="salesDisplay"
              name={t('chart.sales')}
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
              name={t('chart.expense')}
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
          <span className="summary-label">{t('chart.sales')} (12M):</span>
          <span className="summary-value sales">
            ₹{data.reduce((sum, d) => sum + d.sales, 0).toLocaleString('en-IN')}
          </span>
        </div>
        <div className="summary-item">
          <span className="summary-label">{t('chart.expense')} (12M):</span>
          <span className="summary-value expense">
            ₹{data.reduce((sum, d) => sum + d.total_expense, 0).toLocaleString('en-IN')}
          </span>
        </div>
        <div className="summary-item">
          <span className="summary-label">{t('chart.profit')} (12M):</span>
          <span className={`summary-value ${data.reduce((sum, d) => sum + d.net_profit, 0) >= 0 ? 'profit' : 'loss'}`}>
            ₹{data.reduce((sum, d) => sum + d.net_profit, 0).toLocaleString('en-IN')}
          </span>
        </div>
      </div>
    </div>
  );
};

export default MonthlyBarChart;
