import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import MonthlyBarChart from './MonthlyBarChart';

jest.mock('recharts', () => {
  const React = require('react');
  return {
    ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
    BarChart: ({ data, children }: any) => (
      <div data-testid="monthly-bar-chart" data-months={JSON.stringify(data.map((d: any) => `${d.year}-${d.month}`))}>
        {children}
      </div>
    ),
    Bar: () => <div />,
    XAxis: () => <div />,
    YAxis: () => <div />,
    CartesianGrid: () => <div />,
    Tooltip: () => <div />,
    Legend: () => <div />,
    Cell: () => <div />,
  };
});

jest.mock('../../utils/deviceId', () => ({
  getOrCreateDeviceId: () => 'test-device-id',
}));

jest.mock('../../context/AuthContext', () => ({
  useAuth: () => ({ isAuthenticated: true }),
}));

jest.mock('../../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
    language: 'en',
  }),
}));

describe('MonthlyBarChart current month live toggle', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('excludes current calendar month by default and shows quick toggle when current month has data', async () => {
    const now = new Date();
    const curYear = now.getFullYear();
    const curMonth = now.getMonth() + 1;
    const prevDate = new Date(curYear, now.getMonth() - 1, 1);
    const prevYear = prevDate.getFullYear();
    const prevMonth = prevDate.getMonth() + 1;

    global.fetch = jest.fn().mockResolvedValue({
      json: async () => ({
        success: true,
        data: [
          {
            year: prevYear,
            month: prevMonth,
            month_name: 'Prev',
            sales: 100000,
            feed_expense: 50000,
            medicine_expense: 2000,
            labor_expense: 5000,
            other_expense: 1000,
            other_income: 0,
            total_discounts: 0,
            total_tds: 0,
            total_expense: 58000,
            net_profit: 42000,
          },
          {
            year: curYear,
            month: curMonth,
            month_name: 'Current',
            sales: 25000,
            feed_expense: 10000,
            medicine_expense: 0,
            labor_expense: 0,
            other_expense: 0,
            other_income: 0,
            total_discounts: 0,
            total_tds: 0,
            total_expense: 10000,
            net_profit: 15000,
          },
        ],
      }),
    }) as any;

    render(<MonthlyBarChart onMonthClick={jest.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId('monthly-bar-chart')).toBeInTheDocument();
    });

    const initialMonths = JSON.parse(screen.getByTestId('monthly-bar-chart').getAttribute('data-months') || '[]');
    expect(initialMonths).toContain(`${prevYear}-${prevMonth}`);
    expect(initialMonths).not.toContain(`${curYear}-${curMonth}`);

    const toggleBtn = screen.getByRole('button', { name: /Show Current Month/i });
    fireEvent.click(toggleBtn);

    const toggledMonths = JSON.parse(screen.getByTestId('monthly-bar-chart').getAttribute('data-months') || '[]');
    expect(toggledMonths).toContain(`${curYear}-${curMonth}`);
    expect(screen.getByText(/Current Month Included \(Live\)/i)).toBeInTheDocument();
  });
});
