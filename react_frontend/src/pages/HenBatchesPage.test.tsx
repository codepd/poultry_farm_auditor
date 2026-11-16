import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import HenBatchesPage from './HenBatchesPage';
import { henBatchesAPI } from '../services/api';

const mockNavigate = jest.fn();
let mockSearch = '';
jest.mock(
  'react-router-dom',
  () => ({
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: '/hen-batches', search: mockSearch }),
  }),
  { virtual: true }
);

jest.mock('../services/api', () => ({
  __esModule: true,
  default: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
  analyticsAPI: {
    getEnhancedMonthlySummary: jest.fn(),
  },
  henBatchesAPI: {
    getHenBatches: jest.fn(),
    getSalesHistory: jest.fn(),
    createHenBatch: jest.fn(),
  },
}));

jest.mock('../context/AuthContext', () => ({
  useAuth: () => ({
    currentTenant: { tenant_id: 'tenant-1', role: 'OWNER' },
  }),
}));

jest.mock('../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
    tStage: (stage: string) => stage,
    tItem: (item: string) => item,
  }),
}));

jest.mock('../hooks/useTenant', () => ({
  useTenant: () => ({
    tenant: { age_category_chick_max_weeks: 6, age_category_grower_max_weeks: 18, age_category_prelayer_max_weeks: 22 },
    timezone: 'Asia/Kolkata',
    dateFormat: 'dd-MM-yyyy',
  }),
}));

jest.mock('../components/HenBatches/MortalityChart', () => () => <div data-testid="mortality-chart" />);

describe('HenBatchesPage alert prefill flow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockSearch = '';
    (henBatchesAPI.getHenBatches as jest.Mock).mockResolvedValue([]);
    (henBatchesAPI.getSalesHistory as jest.Mock).mockResolvedValue([]);
  });

  test('opens Add New Hen Batch form pre-filled with date_added from query params', async () => {
    mockSearch = '?addBatch=1&date_added=2026-09-27';
    render(<HenBatchesPage />);

    await waitFor(() => {
      expect(screen.getByText('Add New Hen Batch')).toBeInTheDocument();
    });

    const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
    expect(dateInput).not.toBeNull();
    expect(dateInput.value).toBe('2026-09-27');
  });
});
