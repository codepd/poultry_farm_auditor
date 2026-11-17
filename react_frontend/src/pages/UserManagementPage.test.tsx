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
});
