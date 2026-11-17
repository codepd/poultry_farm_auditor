import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import PriceLineChart from './PriceLineChart';
import PriceHistoryPage from '../../pages/PriceHistoryPage';
import api from '../../services/api';

jest.mock('recharts', () => {
  const React = require('react');
  return {
    ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
    LineChart: ({ data, children }: any) => (
      <div data-testid="line-chart" data-chart-points={JSON.stringify(data)}>
        {children}
      </div>
    ),
    Line: ({ name, stroke, strokeDasharray, dataKey }: any) => (
      <div
        data-testid={`chart-line-${dataKey}`}
        data-name={name}
        data-stroke={stroke}
        data-dasharray={strokeDasharray}
      />
    ),
    XAxis: () => <div data-testid="x-axis" />,
    YAxis: () => <div data-testid="y-axis" />,
    CartesianGrid: () => <div data-testid="cartesian-grid" />,
    Tooltip: () => <div data-testid="tooltip" />,
    Legend: () => <div data-testid="legend" />,
  };
});

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

jest.mock('../../context/AuthContext', () => ({
  useAuth: () => ({
    currentTenant: { tenant_id: 'tenant-1', role: 'OWNER' },
  }),
}));

jest.mock('../../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
    tItem: (item: string) => item,
  }),
}));

describe('PriceLineChart Year-on-Year colors and PriceHistoryPage September egg data', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('assigns distinct stroke colors and valid SVG strokeDasharray per item-year in YoY mode', () => {
    const prices = [
      { id: 0, tenant_id: 't1', price_date: '2026-08-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 5.4, created_at: '2026-08-01T00:00:00Z' },
      { id: 0, tenant_id: 't1', price_date: '2026-09-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 5.52, created_at: '2026-09-01T00:00:00Z' },
      { id: 0, tenant_id: 't1', price_date: '2025-08-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 4.8, created_at: '2025-08-01T00:00:00Z' },
      { id: 0, tenant_id: 't1', price_date: '2025-09-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 4.95, created_at: '2025-09-01T00:00:00Z' },
    ];

    render(
      <PriceLineChart
        prices={prices}
        priceType="EGG"
        selectedYears={[2026, 2025]}
      />
    );

    const line2026 = screen.getByTestId('chart-line-LARGE EGG (2026)');
    const line2025 = screen.getByTestId('chart-line-LARGE EGG (2025)');

    expect(line2026.getAttribute('data-stroke')).not.toBe(line2025.getAttribute('data-stroke'));
    expect(line2026.getAttribute('data-dasharray')).toBe('0');
    expect(line2025.getAttribute('data-dasharray')).toBe('6 4');

    const chartPoints = JSON.parse(screen.getByTestId('line-chart').getAttribute('data-chart-points') || '[]');
    const sepPoint = chartPoints.find((pt: any) => pt.period === 9);
    expect(sepPoint['LARGE EGG (2026)']).toBe(5.52);
    expect(sepPoint['LARGE EGG (2025)']).toBe(4.95);
  });

  test('PriceHistoryPage fetches price_type=EGG and price_type=FEED and plots September 2026', async () => {
    (api.get as jest.Mock).mockImplementation((url: string) => {
      if (url === '/prices?price_type=EGG') {
        return Promise.resolve({
          data: {
            success: true,
            data: [
              { id: 0, tenant_id: 't1', price_date: '2026-09-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 5.52, created_at: '2026-09-01T00:00:00Z' },
            ],
          },
        });
      }
      if (url === '/prices?price_type=FEED') {
        return Promise.resolve({
          data: {
            success: true,
            data: [
              { id: 1, tenant_id: 't1', price_date: '2026-09-07', price_type: 'FEED', item_name: 'LAYER MASH', price: 29.5, created_at: '2026-09-07T00:00:00Z' },
            ],
          },
        });
      }
      return Promise.resolve({ data: { success: true, data: [] } });
    });

    render(<PriceHistoryPage />);

    await waitFor(() => {
      expect(api.get).toHaveBeenCalledWith('/prices?price_type=EGG');
      expect(api.get).toHaveBeenCalledWith('/prices?price_type=FEED');
    });

    await waitFor(() => {
      expect(screen.getByTestId('chart-line-LARGE EGG (2026)')).toBeInTheDocument();
    });
  });

  test('defaults to 1 egg line (LARGE EGG) and 1 feed line (LAYER MASH) and switches via per-chart dropdown', async () => {
    (api.get as jest.Mock).mockImplementation((url: string) => {
      if (url === '/prices?price_type=EGG') {
        return Promise.resolve({
          data: {
            success: true,
            data: [
              { id: 1, tenant_id: 't1', price_date: '2026-09-01', price_type: 'EGG', item_name: 'LARGE EGG', price: 5.52, created_at: '2026-09-01T00:00:00Z' },
              { id: 2, tenant_id: 't1', price_date: '2026-09-01', price_type: 'EGG', item_name: 'MEDIUM EGG', price: 5.10, created_at: '2026-09-01T00:00:00Z' },
              { id: 3, tenant_id: 't1', price_date: '2026-09-01', price_type: 'EGG', item_name: 'SMALL EGG', price: 4.50, created_at: '2026-09-01T00:00:00Z' },
            ],
          },
        });
      }
      if (url === '/prices?price_type=FEED') {
        return Promise.resolve({
          data: {
            success: true,
            data: [
              { id: 4, tenant_id: 't1', price_date: '2026-09-07', price_type: 'FEED', item_name: 'LAYER MASH', price: 29.5, created_at: '2026-09-07T00:00:00Z' },
              { id: 5, tenant_id: 't1', price_date: '2026-09-07', price_type: 'FEED', item_name: 'GROWER MASH', price: 31.0, created_at: '2026-09-07T00:00:00Z' },
            ],
          },
        });
      }
      return Promise.resolve({ data: { success: true, data: [] } });
    });

    render(<PriceHistoryPage />);

    await waitFor(() => {
      expect(screen.getByTestId('chart-line-LARGE EGG (2026)')).toBeInTheDocument();
      expect(screen.getByTestId('chart-line-LAYER MASH (2026)')).toBeInTheDocument();
    });

    // Other lines are hidden by default to prevent clutter
    expect(screen.queryByTestId('chart-line-MEDIUM EGG (2026)')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-line-SMALL EGG (2026)')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-line-GROWER MASH (2026)')).not.toBeInTheDocument();

    // Switch Egg dropdown to MEDIUM EGG
    fireEvent.change(screen.getByLabelText('Select Egg Type'), { target: { value: 'MEDIUM EGG' } });
    await waitFor(() => {
      expect(screen.getByTestId('chart-line-MEDIUM EGG (2026)')).toBeInTheDocument();
      expect(screen.queryByTestId('chart-line-LARGE EGG (2026)')).not.toBeInTheDocument();
    });

    // Switch Feed dropdown to ALL
    fireEvent.change(screen.getByLabelText('Select Feed Type'), { target: { value: 'ALL' } });
    await waitFor(() => {
      expect(screen.getByTestId('chart-line-LAYER MASH (2026)')).toBeInTheDocument();
      expect(screen.getByTestId('chart-line-GROWER MASH (2026)')).toBeInTheDocument();
    });
  });

  test('displays Feed Price Alignment panel when /prices/feed-validation returns rows', async () => {
    (api.get as jest.Mock).mockImplementation((url: string) => {
      if (url === '/prices?price_type=EGG') {
        return Promise.resolve({ data: { success: true, data: [] } });
      }
      if (url === '/prices?price_type=FEED') {
        return Promise.resolve({
          data: {
            success: true,
            data: [
              { id: 4, tenant_id: 't1', price_date: '2026-10-01', price_type: 'FEED', item_name: 'LAYER MASH', price: 2250, created_at: '2026-10-01T00:00:00Z' },
            ],
          },
        });
      }
      if (url === '/prices/feed-validation') {
        return Promise.resolve({
          data: {
            success: true,
            data: {
              summary: { total: 2, aligned: 1, mismatch: 1, no_sheet_rate: 0 },
              rows: [
                {
                  transaction_id: 11,
                  transaction_date: '2026-10-05',
                  item_name: 'LAYER MASH',
                  quantity: 1000,
                  amount: 30000,
                  ledger_rate_per_kg: 30.0,
                  sheet_date: '2026-10-01',
                  sheet_bag_rate: 2250,
                  sheet_rate_per_kg: 30.0,
                  diff_per_kg: 0,
                  diff_pct: 0,
                  status: 'ALIGNED',
                },
                {
                  transaction_id: 12,
                  transaction_date: '2026-10-12',
                  item_name: 'GROWER MASH',
                  quantity: 500,
                  amount: 17000,
                  ledger_rate_per_kg: 34.0,
                  sheet_date: '2026-10-01',
                  sheet_bag_rate: 2250,
                  sheet_rate_per_kg: 30.0,
                  diff_per_kg: 4.0,
                  diff_pct: 13.33,
                  status: 'MISMATCH',
                },
              ],
            },
          },
        });
      }
      return Promise.resolve({ data: { success: true, data: [] } });
    });

    render(<PriceHistoryPage />);

    await waitFor(() => {
      expect(screen.getByText('Feed Price Alignment (Ledger vs Rate Sheet)')).toBeInTheDocument();
    });
    expect(screen.getByText('Aligned: 1')).toBeInTheDocument();
    expect(screen.getByText('Mismatch: 1')).toBeInTheDocument();
    expect(screen.getByText('ALIGNED')).toBeInTheDocument();
    expect(screen.getByText('MISMATCH')).toBeInTheDocument();
  });
});
