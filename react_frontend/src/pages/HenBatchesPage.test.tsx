import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
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

    // Changing Date Added must NOT overwrite Age at Date Added with elapsed weeks (avoids double-counting)
    fireEvent.change(dateInput, { target: { value: '2026-09-14' } });
    const numberInputs = document.querySelectorAll('input[type="number"]');
    const ageWeeksInput = numberInputs[2] as HTMLInputElement;
    expect(ageWeeksInput.value).toBe('0');
  });

  test('hides sold batches (current_count === 0) by default and toggles them open', async () => {
    (henBatchesAPI.getHenBatches as jest.Mock).mockResolvedValue([
      {
        id: 1,
        batch_name: 'Active Batch',
        initial_count: 11000,
        current_count: 10918,
        age_weeks: 0,
        age_days: 0,
        date_added: '2026-04-29',
      },
      {
        id: 2,
        batch_name: 'Sold Batch 1',
        initial_count: 10000,
        current_count: 0,
        age_weeks: 80,
        age_days: 0,
        date_added: '2025-10-06',
      },
    ]);

    render(<HenBatchesPage />);

    await waitFor(() => {
      expect(screen.getByText('Active Batch')).toBeInTheDocument();
    });
    expect(screen.queryByText('Sold Batch 1')).not.toBeInTheDocument();

    const toggleBtn = screen.getByText(/batches\.showArchived/i);
    fireEvent.click(toggleBtn);

    expect(screen.getByText('Sold Batch 1')).toBeInTheDocument();
    expect(screen.getByText(/batches\.hideArchived/i)).toBeInTheDocument();
  });
});
