import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { translations, translateItemName, translateStageTag } from './locales/translations';
import { LanguageProvider, useLanguage } from './context/LanguageContext';
import Header from './components/Layout/Header';
import Navbar from './components/Layout/Navbar';
import MonthlyStatistics from './components/Home/MonthlyStatistics';
import DataUploadPage from './pages/DataUploadPage';
import { analyticsAPI, usersAPI, adminAPI } from './services/api';

const mockNavigate = jest.fn();
jest.mock(
  'react-router-dom',
  () => ({
    useNavigate: () => mockNavigate,
    useLocation: () => ({ pathname: '/' }),
    Link: ({ to, children, className }: any) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
  }),
  { virtual: true }
);

let mockCurrentRole = 'OWNER';
const mockUpdateUser = jest.fn();

// Mock AuthContext so components render with configurable role
jest.mock('./context/AuthContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    user: {
      user_id: 1,
      email: 'pradeep@mykolipannai.com',
      full_name: 'Pradeep',
      preferred_language: 'en',
      tenants: [{ tenant_id: '8d7939f7-b716-4eb0-98d4-544c18c8dfb8', name: 'Pradeep Farm', role: mockCurrentRole }],
    },
    currentTenant: {
      tenant_id: '8d7939f7-b716-4eb0-98d4-544c18c8dfb8',
      name: 'Pradeep Farm',
      role: mockCurrentRole,
    },
    logout: jest.fn(),
    updateUser: mockUpdateUser,
  }),
}));

jest.mock('./services/api', () => ({
  analyticsAPI: {
    getEnhancedMonthlySummary: jest.fn(),
    getMonthlyBreakdown: jest.fn(),
  },
  usersAPI: {
    getProfile: jest.fn(),
    updateProfile: jest.fn(),
    changePassword: jest.fn(),
    logoutOtherDevices: jest.fn(),
  },
  adminAPI: {
    listUploads: jest.fn(),
    uploadLedger: jest.fn(),
    getUploadStatus: jest.fn(),
    commitUpload: jest.fn(),
  },
}));

// Exact September 2026 fixture from Pradeep Farm
const SEPTEMBER_2026_FIXTURE = {
  year: 2026,
  month: 9,
  total_eggs_sold: 798060,
  total_egg_price: 4331665.32,
  egg_breakdown: [{ type: 'LARGE EGG', quantity: 798060, amount: 4331665.32 }],
  feed_purchased_tonne: 115.67,
  total_feed_price: 3945532.58,
  layer_feed_tonne: 96.75,
  layer_feed_price: 3299736.22,
  rearing_feed_tonne: 18.92,
  rearing_feed_price: 645796.36,
  feed_breakdown: [
    { type: 'LAYER MASH BULK', quantity: 96.75, amount: 3299736.22, stage: 'LAYER' },
    { type: 'SYLO - PLM', quantity: 18.92, amount: 645796.36, stage: 'PRELAYER' },
  ],
  total_medicines: 96115,
  medicine_breakdown: [],
  other_expenses: 382000,
  labor_expenses: 80000,
  other_income: 0,
  hen_sale_income: 0,
  total_discounts: 38557,
  total_tds: 0,
  total_payments: 500000,
  payment_breakdown: [],
  opening_balance: 1200000,
  closing_balance: 1028574.74,
  flock_operating_profit: 974371.1,
  ledger_net_profit: 328574.74,
  farm_net_profit: -53425.26,
  net_profit: -53425.26,
  estimated_hens: 41551.33,
  actual_head_count: 41551.33,
  using_actual_count: true,
  egg_percentage: 64.02,
  egg_selling_price_per_egg: 5.43,
  layer_feed_cost_per_egg: 4.13,
  all_in_cost_per_egg: 5.54,
  net_profit_per_egg: -0.07,
  eggs_per_kg_layer_feed: 8.25,
  layer_feed_grams_per_hen_per_day: 112.4,
  flock_alerts: [
    {
      code: 'UNREGISTERED_CHICK_BATCH',
      severity: 'warning',
      stage: 'CHICK',
      feed_tonne: 12.5,
      feed_amount: 425000,
      estimated_current_age_weeks: 12,
    },
  ],
};

// Helper component to switch language inside tests
const LanguageToggleHarness: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { language, setLanguage } = useLanguage();
  return (
    <div>
      <button data-testid="switch-en" onClick={() => setLanguage('en')}>
        Switch EN
      </button>
      <button data-testid="switch-ta" onClick={() => setLanguage('ta')}>
        Switch TA
      </button>
      <span data-testid="active-lang">{language}</span>
      {children}
    </div>
  );
};

describe('QA Suite 1: English & Tamil Translation Dictionary Parity', () => {
  test('every English translation key exists in Tamil dictionary and is non-empty', () => {
    const enKeys = Object.keys(translations.en);
    const taKeys = Object.keys(translations.ta);

    expect(taKeys.sort()).toEqual(enKeys.sort());

    for (const key of enKeys) {
      expect(translations.en[key].trim().length).toBeGreaterThan(0);
      expect(translations.ta[key].trim().length).toBeGreaterThan(0);
      // Ensure no stray Kannada/corrupted Unicode block characters (U+0C80–U+0CFF) exist in Tamil strings
      expect(/[\u0C80-\u0CFF]/.test(translations.ta[key])).toBe(false);
    }
  });

  test('translateItemName and translateStageTag localize poultry items accurately in Tamil', () => {
    expect(translateItemName('LARGE EGG', 'en')).toBe('LARGE EGG');
    expect(translateItemName('LARGE EGG', 'ta')).toContain('பெரிய முட்டை');
    expect(translateItemName('LABOR', 'ta')).toContain('வேலையாட்கள் சம்பளம்');
    expect(translateItemName('ELECTRICITY', 'ta')).toContain('மின்சாரக் கட்டணம்');
    expect(translateItemName('EMI', 'ta')).toContain('வங்கி தவணை');
    expect(translateItemName('LAYER MASH BULK', 'ta')).toContain('லேயர் மாஷ் தீவனம்');

    expect(translateStageTag('LAYER', 'ta')).toBe('முட்டை தீவனம்');
    expect(translateStageTag('CHICK', 'ta')).toBe('குஞ்சு தீவனம்');
    expect(translateStageTag('GROWER', 'ta')).toBe('குரோவர் தீவனம்');
  });
});

describe('QA Suite 2: MonthlyStatistics Calculations, Presets & EN/TA Snapshot Parity', () => {
  let clipboardSpy: jest.Mock;

  beforeEach(() => {
    mockCurrentRole = 'OWNER';
    localStorage.clear();
    (analyticsAPI.getEnhancedMonthlySummary as jest.Mock).mockResolvedValue(SEPTEMBER_2026_FIXTURE);
    (usersAPI.getProfile as jest.Mock).mockResolvedValue({ preferred_language: 'en', full_name: 'Pradeep' });
    (usersAPI.updateProfile as jest.Mock).mockResolvedValue({ preferred_language: 'ta' });
    clipboardSpy = jest.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: {
        writeText: clipboardSpy,
      },
    });
  });

  test('produces identical numeric values in English and Tamil copy snapshots across all 3 profit presets and zero U+FFFD characters', async () => {
    render(
      <LanguageProvider>
        <LanguageToggleHarness>
          <MonthlyStatistics year={2026} month={9} />
        </LanguageToggleHarness>
      </LanguageProvider>
    );

    // Wait for September 2026 data to load
    await waitFor(() => {
      expect(screen.getAllByText(/7,98,060/).length).toBeGreaterThan(0);
    });

    // --- PRESET 1: Default 'comprehensive' (Farm Overall) ---
    // Net Profit = 43,31,665.32 + 38,557 - 32,99,736.22 - 6,45,796.36 - 96,115 - 3,82,000 = -53,425.26
    // Net Profit / Egg = -53,425.26 / 798,060 = -0.07
    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(1));
    const enComprehensiveReport: string = clipboardSpy.mock.calls[0][0];

    expect(enComprehensiveReport).toContain('₹-0.07');
    expect(enComprehensiveReport).toContain('₹-53,425.26');
    expect(enComprehensiveReport).not.toContain('\uFFFD');

    // Switch to Tamil and copy the exact same preset
    fireEvent.click(screen.getByTestId('switch-ta'));
    await waitFor(() => expect(screen.getByTestId('active-lang').textContent).toBe('ta'));

    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(2));
    const taComprehensiveReport: string = clipboardSpy.mock.calls[1][0];

    expect(taComprehensiveReport).toContain('₹-0.07');
    expect(taComprehensiveReport).toContain('₹-53,425.26');
    expect(taComprehensiveReport).not.toContain('\uFFFD');
    expect(usersAPI.updateProfile).toHaveBeenCalledWith({ preferred_language: 'ta' });

    // --- PRESET 2: Switch to 'Flock Operating' (🥚 கோழி செயல்பாடு / 🥚 Flock Operating) ---
    // Net Profit = 43,31,665.32 + 38,557 - 32,99,736.22 - 96,115 = +9,74,371.10
    // Net Profit / Egg = 9,74,371.10 / 798,060 = +1.22
    const flockPresetBtn = screen.getByTitle(/Laying Flock Only/i);
    fireEvent.click(flockPresetBtn);

    // Copy in Tamil under Flock Operating preset
    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(3));
    const taFlockReport: string = clipboardSpy.mock.calls[2][0];

    // Switch back to English under Flock Operating preset and copy
    fireEvent.click(screen.getByTestId('switch-en'));
    await waitFor(() => expect(screen.getByTestId('active-lang').textContent).toBe('en'));

    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(4));
    const enFlockReport: string = clipboardSpy.mock.calls[3][0];

    // CRITICAL REGRESSION CHECK: Both Tamil and English must show ₹1.22/egg and ₹9,74,371.1 (never ₹-0.07 vs ₹1.22!)
    expect(taFlockReport).toContain('₹1.22');
    expect(enFlockReport).toContain('₹1.22');
    expect(taFlockReport).toContain('₹9,74,371.1');
    expect(enFlockReport).toContain('₹9,74,371.1');
    expect(taFlockReport).toContain('(22.3%)');
    expect(enFlockReport).toContain('(22.3%)');
    expect(taFlockReport).not.toContain('\uFFFD');
    expect(enFlockReport).not.toContain('\uFFFD');

    // In Flock Operating preset, Overheads (₹3,82,000) are excluded from profit and should not appear as an active deduction line
    expect(taFlockReport).not.toContain('₹3,82,000');
    expect(enFlockReport).not.toContain('₹3,82,000');

    // --- PRESET 3: Switch to 'Statement Ledger' (📑 கம்பெனி லெட்ஜர் / 📑 Statement Ledger) ---
    // Net Profit = 43,31,665.32 + 38,557 - 39,45,532.58 - 96,115 = +3,28,574.74
    // Net Profit / Egg = 3,28,574.74 / 798,060 = +0.41
    const ledgerPresetBtn = screen.getByTitle(/Ledger Statement:/i);
    fireEvent.click(ledgerPresetBtn);

    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(5));
    const enLedgerReport: string = clipboardSpy.mock.calls[4][0];

    fireEvent.click(screen.getByTestId('switch-ta'));
    fireEvent.click(screen.getByTitle('Copy Monthly Summary to Clipboard'));
    await waitFor(() => expect(clipboardSpy).toHaveBeenCalledTimes(6));
    const taLedgerReport: string = clipboardSpy.mock.calls[5][0];

    expect(enLedgerReport).toContain('₹0.41');
    expect(taLedgerReport).toContain('₹0.41');
    expect(enLedgerReport).toContain('₹3,28,574.74');
    expect(taLedgerReport).toContain('₹3,28,574.74');
  });

  test('renders Smart Flock Alert banner in both English and Tamil and navigates to /hen-batches', async () => {
    render(
      <LanguageProvider>
        <LanguageToggleHarness>
          <MonthlyStatistics year={2026} month={9} />
        </LanguageToggleHarness>
      </LanguageProvider>
    );

    await waitFor(() => {
      expect(
        screen.getByText(/Unregistered Chick Batch Detected from Feed Purchases/i)
      ).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/Go to Hen Batches/i));
    expect(mockNavigate).toHaveBeenCalledWith('/hen-batches');

    fireEvent.click(screen.getByTestId('switch-ta'));
    await waitFor(() => {
      expect(
        screen.getByText(/புதிய கோழிக் குஞ்சு பேட்ச் தீவனக் கணக்கில் கண்டறியப்பட்டுள்ளது/i)
      ).toBeInTheDocument();
    });
  });
});

describe('QA Suite 3: Header Language Switcher, Navbar Role-Based Tabs & S3 Vault Search', () => {
  beforeEach(() => {
    mockCurrentRole = 'OWNER';
    localStorage.clear();
    (usersAPI.getProfile as jest.Mock).mockResolvedValue({ preferred_language: 'en', full_name: 'Pradeep' });
    (usersAPI.updateProfile as jest.Mock).mockResolvedValue({ preferred_language: 'ta' });
    (adminAPI.listUploads as jest.Mock).mockResolvedValue([
      {
        id: 101,
        tenant_id: '8d7939f7-b716-4eb0-98d4-544c18c8dfb8',
        s3_bucket: 'kolipannai-dev-documents',
        s3_key: 'tenants/8d7939f7/ledgers/2026/09/SEP_2026_SAKTHI.pdf',
        file_name: 'SEP_2026_SAKTHI.pdf',
        file_type: 'LEDGER_PDF',
        file_size: 245760,
        sha256_hash: 'abcdef1234567890abcdef1234567890',
        document_year: 2026,
        document_month: 9,
        upload_source: 'WEB_PORTAL',
        upload_status: 'COMMITTED',
        created_at: '2026-10-02T10:00:00Z',
        download_url: 'https://kolipannai-dev-documents.s3.amazonaws.com/SEP_2026_SAKTHI.pdf',
      },
      {
        id: 102,
        tenant_id: '8d7939f7-b716-4eb0-98d4-544c18c8dfb8',
        s3_bucket: 'kolipannai-dev-documents',
        s3_key: 'tenants/8d7939f7/feed-prices/2025/FEED_RATE_CARD_2025.jpg',
        file_name: 'FEED_RATE_CARD_2025.jpg',
        file_type: 'FEED_PRICE_IMAGE',
        file_size: 102400,
        sha256_hash: '9876543210fedcba9876543210fedcba',
        document_year: 2025,
        document_month: 4,
        upload_source: 'S3_MIGRATION',
        upload_status: 'ARCHIVED',
        created_at: '2025-04-15T10:00:00Z',
      },
    ]);
  });

  test('Header EN | தமிழ் toggle switches Navbar labels and persists to user profile', async () => {
    render(
      <LanguageProvider>
        <Header />
        <Navbar />
      </LanguageProvider>
    );

    expect(screen.getAllByText('Home').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Expenses').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Hen Batches').length).toBeGreaterThan(0);

    // Click தமிழ் button in Header
    fireEvent.click(screen.getByText('தமிழ்'));

    await waitFor(() => {
      expect(screen.getAllByText('முகப்பு').length).toBeGreaterThan(0);
      expect(screen.getAllByText('செலவுகள்').length).toBeGreaterThan(0);
      expect(screen.getAllByText('கோழிப் பேட்ச்கள்').length).toBeGreaterThan(0);
    });

    expect(localStorage.getItem('kolipannai_lang')).toBe('ta');
    expect(usersAPI.updateProfile).toHaveBeenCalledWith({ preferred_language: 'ta' });
  });

  test('Navbar hides Expenses, Upload Sheets, and Team tabs for VIEWER/AUDITOR role', () => {
    mockCurrentRole = 'VIEWER';
    render(
      <LanguageProvider>
        <Navbar />
      </LanguageProvider>
    );

    expect(screen.getAllByText('Home').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Hen Batches').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Price History').length).toBeGreaterThan(0);
    expect(screen.queryByText('Expenses')).not.toBeInTheDocument();
    expect(screen.queryByText('Upload Sheets')).not.toBeInTheDocument();
    expect(screen.queryByText('Team')).not.toBeInTheDocument();
  });

  test('DataUploadPage S3 Audit Vault filters documents by filename/S3 key search query', async () => {
    render(
      <LanguageProvider>
        <DataUploadPage />
      </LanguageProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('SEP_2026_SAKTHI.pdf')).toBeInTheDocument();
      expect(screen.getByText('FEED_RATE_CARD_2025.jpg')).toBeInTheDocument();
    });

    const searchInput = screen.getByPlaceholderText(/Search file name or S3 key/i);
    fireEvent.change(searchInput, { target: { value: 'SAKTHI' } });

    expect(screen.getByText('SEP_2026_SAKTHI.pdf')).toBeInTheDocument();
    expect(screen.queryByText('FEED_RATE_CARD_2025.jpg')).not.toBeInTheDocument();
  });
});
