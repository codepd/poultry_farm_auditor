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
  },
}));

jest.mock('../context/LanguageContext', () => ({
  useLanguage: () => ({
    t: (key: string) => key,
  }),
}));

describe('UserManagementPage promote/demote/remove actions', () => {
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
    ]);
    (usersAPI.getInvitations as jest.Mock).mockResolvedValue([]);
    (usersAPI.updateUserRole as jest.Mock).mockResolvedValue({ success: true });
    (usersAPI.removeUser as jest.Mock).mockResolvedValue({ success: true });
  });

  test('promotes/demotes a non-owner member and removes a member after confirmation', async () => {
    const confirmSpy = jest.spyOn(window, 'confirm').mockReturnValue(true);

    render(<UserManagementPage />);

    await waitFor(() => {
      expect(screen.getByText('Suresh Helper')).toBeInTheDocument();
    });

    const roleSelect = screen.getByLabelText('Role for Suresh Helper') as HTMLSelectElement;
    expect(roleSelect.value).toBe('OTHER_USER');

    fireEvent.change(roleSelect, { target: { value: 'MANAGER' } });
    await waitFor(() => {
      expect(usersAPI.updateUserRole).toHaveBeenCalledWith(2, 'MANAGER');
    });

    fireEvent.click(screen.getByText('Remove'));
    await waitFor(() => {
      expect(usersAPI.removeUser).toHaveBeenCalledWith(2);
    });

    confirmSpy.mockRestore();
  });
});
