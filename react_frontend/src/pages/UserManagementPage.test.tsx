import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import UserManagementPage from './UserManagementPage';
import { authAPI, usersAPI } from '../services/api';

jest.mock('../services/api', () => ({
  __esModule: true,
  default: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn() },
  authAPI: {
    getCountryCodes: jest.fn(),
  },
  usersAPI: {
    getUsers: jest.fn(),
    getInvitations: jest.fn(),
    inviteUser: jest.fn(),
    generateMagicLink: jest.fn(),
    updateUserRole: jest.fn(),
    removeUser: jest.fn(),
    transferOwnership: jest.fn(),
    revokeInvitation: jest.fn(),
  },
}));

jest.mock('../context/AuthContext', () => ({
  useAuth: () => ({
    user: { user_id: 1, email: 'owner@mykolipannai.com' },
    currentTenant: { tenant_id: 'tenant-1', role: 'OWNER' },
  }),
}));

jest.mock('../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
  }),
}));

describe('UserManagementPage promote/demote/remove/transfer/revoke actions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (authAPI.getCountryCodes as jest.Mock).mockResolvedValue([]);
    (usersAPI.getUsers as jest.Mock).mockResolvedValue([
      {
        id: 1,
        email: 'owner@mykolipannai.com',
        phone: '+919000000001',
        full_name: 'Pradeep Owner',
        is_active: true,
        role: 'OWNER',
        is_owner: true,
      },
      {
        id: 2,
        email: 'helper@mykolipannai.com',
        phone: '+919000000002',
        full_name: 'Suresh Helper',
        is_active: true,
        role: 'OTHER_USER',
        is_owner: false,
      },
      {
        id: 3,
        email: 'coowner@mykolipannai.com',
        phone: '+919000000003',
        full_name: 'Karthik CoOwner',
        is_active: true,
        role: 'CO_OWNER',
        is_owner: false,
      },
    ]);
    (usersAPI.getInvitations as jest.Mock).mockResolvedValue([
      {
        id: 10,
        role: 'MANAGER',
        phone: { String: '+919000000099', Valid: true },
        expires_at: '2026-12-31T00:00:00Z',
      },
    ]);
    (usersAPI.updateUserRole as jest.Mock).mockResolvedValue({ success: true });
    (usersAPI.removeUser as jest.Mock).mockResolvedValue({ success: true });
    (usersAPI.transferOwnership as jest.Mock).mockResolvedValue({ success: true });
    (usersAPI.revokeInvitation as jest.Mock).mockResolvedValue({ success: true });
  });

  test('promotes/demotes, transfers ownership to Co-Owner, removes member, and cancels pending invite', async () => {
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);

    render(<UserManagementPage />);

    await waitFor(() => {
      expect(screen.getByText('Suresh Helper')).toBeInTheDocument();
      expect(screen.getByText('Karthik CoOwner')).toBeInTheDocument();
    });

    const roleSelect = screen.getByLabelText('Role for Suresh Helper') as HTMLSelectElement;
    expect(roleSelect.value).toBe('OTHER_USER');

    fireEvent.change(roleSelect, { target: { value: 'MANAGER' } });
    await waitFor(() => {
      expect(usersAPI.updateUserRole).toHaveBeenCalledWith(2, 'MANAGER');
    });

    fireEvent.click(screen.getByText('Make Owner'));
    await waitFor(() => {
      expect(usersAPI.transferOwnership).toHaveBeenCalledWith(3, 'CO_OWNER');
    });

    fireEvent.click(screen.getAllByText('Remove')[0]);
    await waitFor(() => {
      expect(usersAPI.removeUser).toHaveBeenCalledWith(2);
    });

    fireEvent.click(screen.getByText('Cancel'));
    await waitFor(() => {
      expect(usersAPI.revokeInvitation).toHaveBeenCalledWith(10);
    });

    confirmSpy.mockRestore();
  });

  test('allows Owner to toggle View Profit checkbox per Manager', async () => {
    (usersAPI.getUsers as jest.Mock).mockResolvedValue([
      { id: 1, full_name: 'Owner', email: 'owner@example.com', phone: '', role: 'OWNER', is_owner: true, is_active: true, created_at: '', updated_at: '' },
      { id: 3, full_name: 'Manager One', email: '', phone: '+919999999991', role: 'MANAGER', is_owner: false, can_view_sensitive_data: false, is_active: true, created_at: '', updated_at: '' },
      { id: 4, full_name: 'Manager Two', email: '', phone: '+919999999992', role: 'MANAGER', is_owner: false, can_view_sensitive_data: true, is_active: true, created_at: '', updated_at: '' },
    ]);
    (usersAPI as any).updateUserSensitiveAccess = jest.fn().mockResolvedValue({
      user_id: 3,
      can_view_sensitive_data: true,
    });

    render(<UserManagementPage />);
    const m1Checkbox = (await screen.findByLabelText('View Profit for Manager One')) as HTMLInputElement;
    const m2Checkbox = (await screen.findByLabelText('View Profit for Manager Two')) as HTMLInputElement;
    expect(m1Checkbox.checked).toBe(false);
    expect(m2Checkbox.checked).toBe(true);

    fireEvent.click(m1Checkbox);
    await waitFor(() => {
      expect((usersAPI as any).updateUserSensitiveAccess).toHaveBeenCalledWith(3, true);
    });
  });

  test('allows Owner to open Permissions panel and toggle granular permissions per member', async () => {
    (usersAPI.getUsers as jest.Mock).mockResolvedValue([
      { id: 1, full_name: 'Owner', email: 'owner@example.com', phone: '', role: 'OWNER', is_owner: true, is_active: true, created_at: '', updated_at: '' },
      { id: 3, full_name: 'Manager One', email: '', phone: '+919999999991', role: 'MANAGER', is_owner: false, can_view_sensitive_data: false, permissions: ['manage:expense', 'submit:daily_log'], is_active: true, created_at: '', updated_at: '' },
    ]);
    (usersAPI as any).updateUserPermissions = jest.fn().mockResolvedValue({
      user_id: 3,
      permissions: ['manage:expense', 'submit:daily_log', 'manage:ledger'],
      can_view_sensitive_data: false,
    });

    render(<UserManagementPage />);
    const permsBtn = await screen.findByText('Permissions');
    fireEvent.click(permsBtn);

    const panel = screen.getByTestId('permissions-panel-3');
    expect(panel).toBeInTheDocument();

    // manage:expense implies read:expense so Read Expenses checkbox is checked & disabled
    const readExpensesCheckbox = screen.getByLabelText('Read Expenses') as HTMLInputElement;
    expect(readExpensesCheckbox.checked).toBe(true);
    expect(readExpensesCheckbox.disabled).toBe(true);

    // Toggle Manage Ledger / Sheets
    fireEvent.click(screen.getByLabelText('Manage Ledger / Sheets'));
    await waitFor(() => {
      expect((usersAPI as any).updateUserPermissions).toHaveBeenCalledWith(3, [
        'manage:expense',
        'submit:daily_log',
        'manage:ledger',
      ]);
    });
  });
});
