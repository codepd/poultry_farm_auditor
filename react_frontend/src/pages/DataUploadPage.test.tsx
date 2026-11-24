import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import DataUploadPage from './DataUploadPage';
import { adminAPI } from '../services/api';

const mockNavigate = jest.fn();
jest.mock(
  'react-router-dom',
  () => ({
    useNavigate: () => mockNavigate,
  }),
  { virtual: true }
);

jest.mock('../services/api', () => ({
  __esModule: true,
  default: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn() },
  adminAPI: {
    listUploads: jest.fn(),
    reconcileUpload: jest.fn(),
    commitUpload: jest.fn(),
    uploadLedger: jest.fn(),
    openUploadFile: jest.fn(),
  },
}));

jest.mock('../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
  }),
}));

describe('DataUploadPage Import Mode selection', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (adminAPI.commitUpload as jest.Mock).mockResolvedValue({ success: true });
    (adminAPI.listUploads as jest.Mock).mockResolvedValue([
      {
        id: 42,
        file_name: '26_10_OCT_revised.pdf',
        file_type: 'LEDGER_PDF',
        document_year: 2026,
        document_month: 10,
        upload_source: 'UI',
        upload_status: 'PARSED',
        created_at: '2026-11-15T00:00:00Z',
        parse_result: {
          year: 2026,
          month: 10,
          pdf_filename: '26_10_OCT_revised.pdf',
          summary: {
            feed_provider: 'SAKTHI POULTRY FEEDS',
            ledger_period: '1-Oct-26 to 31-Oct-26',
            total_eggs: 5531150.85,
            total_feeds: 4559423.58,
            total_medicines: 69801,
            total_discounts: 82526,
            total_payments: 500000,
            net_profit: 901926.27,
            validation_difference: 0,
          },
          feeds: [],
          eggs: [],
          discounts: [],
          payments: [],
        },
      },
    ]);
  });

  test('defaults to append_file and passes replace_month when selected', async () => {
    render(<DataUploadPage />);

    await waitFor(() => {
      expect(screen.getByText('26_10_OCT_revised.pdf')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/^(Review|Inspect|upload\.reviewBtn)$/i));

    const appendRadio = screen.getByLabelText(/Add \/ Keep other files for this month/i) as HTMLInputElement;
    const replaceRadio = screen.getByLabelText(/Replace all ledger imports for this month/i) as HTMLInputElement;

    expect(appendRadio.checked).toBe(true);
    expect(replaceRadio.checked).toBe(false);

    fireEvent.click(replaceRadio);
    expect(replaceRadio.checked).toBe(true);

    fireEvent.click(screen.getByText(/Approve & Save to Farm Records/i));

    await waitFor(() => {
      expect(adminAPI.commitUpload).toHaveBeenCalledWith(42, 'replace_month');
    });
  });

  test('displays extracted feed prices and monthly alignment on FEED_PRICE_IMAGE upload and allows committing', async () => {
    (adminAPI.uploadLedger as jest.Mock)
      .mockResolvedValueOnce({
        upload_id: 99,
        file_name: '06_01_2026.jpeg',
        file_type: 'FEED_PRICE_IMAGE',
        upload_status: 'PREVIEWED',
        price_date: '2026-06-01',
        feed_prices: { 'LAYER MASH': 2250 },
        validation: {
          summary: { total: 1, aligned: 1, mismatch: 0, no_sheet_rate: 0 },
          rows: [
            {
              transaction_id: 201,
              transaction_date: '2026-06-05',
              item_name: 'LAYER MASH',
              ledger_rate_per_kg: 30.0,
              sheet_rate_per_kg: 30.0,
              diff_per_kg: 0,
              status: 'ALIGNED',
            },
          ],
        },
      })
      .mockResolvedValueOnce({
        upload_id: 99,
        file_name: '06_01_2026.jpeg',
        file_type: 'FEED_PRICE_IMAGE',
        upload_status: 'COMMITTED',
        price_date: '2026-06-01',
        feed_prices: { 'LAYER MASH': 2250 },
        validation: {
          summary: { total: 1, aligned: 1, mismatch: 0, no_sheet_rate: 0 },
          rows: [
            {
              transaction_id: 201,
              transaction_date: '2026-06-05',
              item_name: 'LAYER MASH',
              ledger_rate_per_kg: 30.0,
              sheet_rate_per_kg: 30.0,
              diff_per_kg: 0,
              status: 'ALIGNED',
            },
          ],
        },
      });

    const { container } = render(<DataUploadPage />);
    const input = container.querySelector('#file-upload-input') as HTMLInputElement;
    const file = new File(['img'], '06_01_2026.jpeg', { type: 'image/jpeg' });
    fireEvent.change(input, { target: { files: [file] } });

    fireEvent.click(screen.getByText(/upload\.submitBtn/i));

    await waitFor(() => {
      expect(screen.getByText('Feed Price Card Extracted — Ready to Save')).toBeInTheDocument();
    });
    expect(screen.getByText('₹2250.00')).toBeInTheDocument();
    expect(screen.getByText('ALIGNED')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Confirm & Save to Price History'));

    await waitFor(() => {
      expect(adminAPI.uploadLedger).toHaveBeenCalledWith(file, true);
      expect(screen.getByText('Feed Prices Saved & Validated!')).toBeInTheDocument();
    });
  });

  test('renders Daily Log vs Ledger Reconciliation and passes supersede_daily_log_ids on commit', async () => {
    (adminAPI.reconcileUpload as jest.Mock).mockResolvedValue({
      summary: {
        exact_match: 0,
        date_shift: 1,
        mismatch: 0,
        ledger_only: 0,
        daily_log_only: 0,
      },
      matches: [
        {
          status: 'DATE_SHIFT',
          color: 'yellow',
          date_diff_days: 1,
          default_resolution: 'KEEP_LEDGER',
          ledger_item: { date: '2026-10-18', category: 'EGG', item_name: 'LARGE EGG', quantity: 1000, amount: 5500, source: 'LEDGER' },
          daily_log_item: { id: 901, date: '2026-10-17', category: 'EGG', item_name: 'LARGE EGG', quantity: 1000, amount: 5500, source: 'DAILY_LOG' },
        },
      ],
    });

    render(<DataUploadPage />);
    await waitFor(() => {
      expect(screen.getByText('26_10_OCT_revised.pdf')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/^(Review|Inspect|upload\.reviewBtn)$/i));

    await waitFor(() => {
      expect(screen.getByText('Daily Log vs Ledger Reconciliation')).toBeInTheDocument();
    });
    expect(screen.getByText('DATE SHIFT (+1d)')).toBeInTheDocument();

    fireEvent.click(screen.getByText(/Approve & Save to Farm Records/i));
    await waitFor(() => {
      expect(adminAPI.commitUpload).toHaveBeenCalledWith(42, 'append_file', [901]);
    });
  });

  test('streams S3 ledger file via authenticated openUploadFile on View File click', async () => {
    (adminAPI.openUploadFile as jest.Mock).mockResolvedValue(undefined);
    render(<DataUploadPage />);

    const viewBtn = await screen.findByText(/upload\.viewFileBtn/i);
    fireEvent.click(viewBtn);

    await waitFor(() => {
      expect(adminAPI.openUploadFile).toHaveBeenCalledWith(42);
    });
  });
});
