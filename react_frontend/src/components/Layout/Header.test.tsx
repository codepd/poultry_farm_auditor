import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Header from './Header';
import { dailyLogsAPI } from '../../services/api';

jest.mock('../../services/api', () => ({
  __esModule: true,
  default: { get: jest.fn(), post: jest.fn() },
  usersAPI: {
    updateProfile: jest.fn(),
    changePassword: jest.fn(),
    logoutOtherDevices: jest.fn(),
  },
  dailyLogsAPI: {
    list: jest.fn(),
    approve: jest.fn(),
    reject: jest.fn(),
  },
}));

jest.mock('../../context/AuthContext', () => ({
  useAuth: () => ({
    currentTenant: { id: 't1', name: 'Pradeep Farm', role: 'OWNER' },
    user: { id: 1, full_name: 'Pradeep', email: 'p@example.com', must_set_password: false },
    logout: jest.fn(),
    updateUser: jest.fn(),
  }),
}));

jest.mock('../../context/LanguageContext', () => ({
  useLanguage: () => ({
    language: 'en',
    setLanguage: jest.fn(),
    t: (k: string) => k,
  }),
}));

describe('Header pending approval notifications', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (dailyLogsAPI.list as jest.Mock).mockResolvedValue([
      {
        id: 77,
        tenant_id: 't1',
        log_type: 'EGG_SALE',
        log_date: '2026-11-18',
        category: 'EGG',
        item_name: 'LARGE EGG',
        quantity: 1000,
        unit: 'NOS',
        rate: 5.5,
        amount: 5500,
        status: 'PENDING_APPROVAL',
        submitted_by_user_id: 4,
        submitted_by_name: 'Ramesh',
        created_at: '2026-11-18T10:00:00Z',
      },
    ]);
    (dailyLogsAPI.approve as jest.Mock).mockResolvedValue({});
  });

  test('shows notification badge count and lets Owner/Manager approve from header dropdown', async () => {
    render(<Header />);

    await waitFor(() => {
      expect(screen.getByTestId('notification-badge')).toHaveTextContent('1');
    });

    fireEvent.click(screen.getByLabelText('Pending approval notifications'));
    expect(screen.getByText('Pending Approvals (1)')).toBeInTheDocument();
    expect(screen.getByText(/By Ramesh/i)).toBeInTheDocument();

    fireEvent.click(screen.getByText('Approve'));
    await waitFor(() => {
      expect(dailyLogsAPI.approve).toHaveBeenCalledWith(77);
    });
  });
});
