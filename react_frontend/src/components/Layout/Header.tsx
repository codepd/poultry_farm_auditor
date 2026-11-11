import React, { useState, useEffect } from 'react';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';
import { usersAPI } from '../../services/api';
import './Header.css';

const Header: React.FC = () => {
  const { currentTenant, user, logout, updateUser } = useAuth();
  const { language, setLanguage, t } = useLanguage();
  const [showProfileMenu, setShowProfileMenu] = useState(false);
  const [editingName, setEditingName] = useState(false);
  const [nameInput, setNameInput] = useState('');
  const [changingPassword, setChangingPassword] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [logoutOtherAfterPasswordChange, setLogoutOtherAfterPasswordChange] = useState(true);
  const [saving, setSaving] = useState(false);

  const needsName = !user?.full_name;

  const getInitials = (name?: string) => {
    if (!name) return 'U';
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest('.profile-menu')) {
        setShowProfileMenu(false);
        setEditingName(false);
      }
    };

    if (showProfileMenu) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showProfileMenu]);

  const handleSaveName = async () => {
    if (!nameInput.trim()) return;
    setSaving(true);
    try {
      await usersAPI.updateProfile({ full_name: nameInput.trim() });
      updateUser({ full_name: nameInput.trim() });
      setEditingName(false);
      setShowProfileMenu(false);
    } catch {
      // ignore
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    if (!newPassword.trim() || newPassword.trim().length < 6) {
      alert('New password must be at least 6 characters');
      return;
    }
    setSaving(true);
    try {
      const result = await usersAPI.changePassword({
        current_password: currentPassword || undefined,
        new_password: newPassword.trim(),
        logout_other_devices: logoutOtherAfterPasswordChange,
      });
      setChangingPassword(false);
      setCurrentPassword('');
      setNewPassword('');
      alert(result?.message || 'Password updated');
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to change password');
    } finally {
      setSaving(false);
    }
  };

  const handleLogoutOtherDevices = async () => {
    setSaving(true);
    try {
      const result = await usersAPI.logoutOtherDevices();
      alert(`Logged out ${result?.data?.revoked_sessions ?? 0} other device sessions`);
    } catch (err: any) {
      alert(err.response?.data?.error || 'Failed to logout other devices');
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      {needsName && (
        <div className="profile-banner" onClick={() => { setShowProfileMenu(true); setEditingName(true); setNameInput(''); }}>
          {t('header.setupName')}
        </div>
      )}
      <header className="app-header">
        <div className="header-left">
          <h1 className="farm-name">{currentTenant?.name || 'Farm'}</h1>
        </div>
        <div className="header-right">
          <div className="lang-switcher" role="group" aria-label={t('header.languageLabel')}>
            <button
              type="button"
              className={`lang-btn ${language === 'en' ? 'active' : ''}`}
              onClick={() => setLanguage('en')}
              title="Switch to English (Saved to your profile)"
            >
              EN
            </button>
            <button
              type="button"
              className={`lang-btn ${language === 'ta' ? 'active' : ''}`}
              onClick={() => setLanguage('ta')}
              title="தமிழில் மாற்று (உங்கள் கணக்கில் சேமிக்கப்படும்)"
            >
              தமிழ்
            </button>
          </div>

          <div className="profile-menu" onClick={() => setShowProfileMenu(!showProfileMenu)}>
            <div className={`profile-icon ${needsName ? 'needs-attention' : ''}`}>
              {user?.full_name ? getInitials(user.full_name) : 'U'}
            </div>
            {showProfileMenu && (
              <div className="profile-dropdown" onClick={e => e.stopPropagation()}>
                <div className="profile-info">
                  {editingName ? (
                    <div className="name-edit-form">
                      <input
                        type="text"
                        value={nameInput}
                        onChange={e => setNameInput(e.target.value)}
                        placeholder="Your full name"
                        className="name-input"
                        autoFocus
                        onKeyDown={e => { if (e.key === 'Enter') handleSaveName(); }}
                      />
                      <div className="name-edit-actions">
                        <button onClick={handleSaveName} disabled={saving || !nameInput.trim()} className="save-name-btn">
                          {saving ? t('header.saving') : t('header.save')}
                        </button>
                        <button onClick={() => setEditingName(false)} className="cancel-name-btn">{t('header.cancel')}</button>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="profile-name">
                        {user?.full_name || 'User'}
                        <button className="edit-name-btn" onClick={(e) => { e.stopPropagation(); setEditingName(true); setNameInput(user?.full_name || ''); }}>
                          {t('header.edit')}
                        </button>
                      </div>
                      <div className="profile-email">{user?.email}</div>
                    </>
                  )}
                  {currentTenant && !editingName && (
                    <div className="profile-role">
                      <span className="role-badge">{currentTenant.role}</span>
                    </div>
                  )}
                </div>
                <div className="profile-actions">
                  {!changingPassword ? (
                    <>
                      <button
                        onClick={() => {
                          setChangingPassword(true);
                          setCurrentPassword('');
                          setNewPassword('');
                          setLogoutOtherAfterPasswordChange(true);
                        }}
                        className="secondary-action-btn"
                        disabled={saving}
                      >
                        {t('header.changePassword')}
                      </button>
                      <button onClick={handleLogoutOtherDevices} className="secondary-action-btn" disabled={saving}>
                        {t('header.logoutOtherDevices')}
                      </button>
                    </>
                  ) : (
                    <div className="password-change-form">
                      <input
                        type="password"
                        value={currentPassword}
                        onChange={(e) => setCurrentPassword(e.target.value)}
                        placeholder="Current password (if set)"
                        className="password-input"
                        disabled={saving}
                      />
                      <input
                        type="password"
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        placeholder="New password"
                        className="password-input"
                        disabled={saving}
                      />
                      <label className="password-option-row">
                        <input
                          type="checkbox"
                          checked={logoutOtherAfterPasswordChange}
                          onChange={(e) => setLogoutOtherAfterPasswordChange(e.target.checked)}
                          disabled={saving}
                        />
                        <span>Logout other devices</span>
                      </label>
                      <div className="password-actions">
                        <button onClick={handleChangePassword} className="save-name-btn" disabled={saving}>
                          {saving ? t('header.saving') : t('header.save')}
                        </button>
                        <button onClick={() => setChangingPassword(false)} className="cancel-name-btn" disabled={saving}>
                          {t('header.cancel')}
                        </button>
                      </div>
                    </div>
                  )}
                  <button onClick={logout} className="logout-btn">
                    {t('header.logout')}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </header>
    </>
  );
};

export default Header;

