import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import ExpensesPage from './ExpensesPage';
import api, { analyticsAPI } from '../services/api';

jest.mock('../services/api', () => {
  const mockApi = {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  };
  return {
    __esModule: true,
    default: mockApi,
    analyticsAPI: {
      getEnhancedMonthlySummary: jest.fn(),
    },
    transactionsAPI: {
      create: jest.fn(),
      update: jest.fn(),
      delete: jest.fn(),
    },
  };
});

jest.mock('../context/AuthContext', () => ({
  useAuth: () => ({
    currentTenant: { id: 'tenant-1', name: 'Test Farm', role: 'OWNER' },
  }),
}));

jest.mock('../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
    tItem: (item: string) => item,
    formatMonthYear: (year: number, month: number) => `${year}-${String(month).padStart(2, '0')}`,
    language: 'en',
  }),
}));

describe('ExpensesPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (analyticsAPI.getEnhancedMonthlySummary as jest.Mock).mockResolvedValue({
      year: new Date().getFullYear(),
      month: new Date().getMonth() + 1,
      other_expenses: 500,
    });
  });

  it('renders without crashing when item_name and notes are Go sql.NullString objects', async () => {
    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = String(now.getMonth() + 1).padStart(2, '0');
    const txDate = `${currentYear}-${currentMonth}-15T00:00:00Z`;

    (api.get as jest.Mock).mockResolvedValue({
      data: {
        success: true,
        data: [
          {
            id: 101,
            transaction_date: txDate,
            category: 'OTHER',
            item_name: { String: 'Discount', Valid: true },
            amount: 500,
            notes: { String: 'Ledger import: b8bb15996f_26_10_OCT.pdf', Valid: true },
            status: 'APPROVED',
          },
          {
            id: 102,
            transaction_date: txDate,
            category: 'OTHER',
            item_name: { String: 'Labor', Valid: true },
            amount: 1200,
            notes: { String: '', Valid: false },
            status: 'APPROVED',
          },
        ],
      },
    });

    render(<ExpensesPage />);

    await waitFor(() => {
      expect(screen.getAllByText('Discount').length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText('Ledger import: b8bb15996f_26_10_OCT.pdf').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Labor').length).toBeGreaterThan(0);
  });
});
