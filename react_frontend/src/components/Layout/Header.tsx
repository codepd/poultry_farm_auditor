import React, { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';
import { usersAPI, dailyLogsAPI, DailyLogSubmission } from '../../services/api';
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

  // Mandatory password setup state when user logs in via WhatsApp Magic Link
  const [forcedFullName, setForcedFullName] = useState(user?.full_name || '');
  const [forcedPassword, setForcedPassword] = useState('');
  const [forcedConfirmPassword, setForcedConfirmPassword] = useState('');
  const [forcedError, setForcedError] = useState('');

  // PWA Install Prompt state
  const [deferredInstallPrompt, setDeferredInstallPrompt] = useState<any>(null);
  const [showInstallGuide, setShowInstallGuide] = useState(false);
  const [isStandalonePWA, setIsStandalonePWA] = useState(false);

  // Pending Approvals Notification Bell (Owner / Co-Owner / Admin / Manager)
  const canApproveDailyLogs = Boolean(
    currentTenant && ['OWNER', 'CO_OWNER', 'ADMIN', 'MANAGER'].includes(currentTenant.role)
  );
  const [pendingApprovals, setPendingApprovals] = useState<DailyLogSubmission[]>([]);
  const [showNotifications, setShowNotifications] = useState(false);

  const loadPendingApprovals = useCallback(() => {
    if (!canApproveDailyLogs || !dailyLogsAPI?.list) {
      setPendingApprovals([]);
      return;
    }
    dailyLogsAPI
      .list('PENDING_APPROVAL')
      .then((list) => setPendingApprovals(list))
      .catch(() => {});
  }, [canApproveDailyLogs]);

  useEffect(() => {
    loadPendingApprovals();
    window.addEventListener('transactionsUpdated', loadPendingApprovals);
    const interval = setInterval(loadPendingApprovals, 60000);
    return () => {
      window.removeEventListener('transactionsUpdated', loadPendingApprovals);
      clearInterval(interval);
    };
  }, [loadPendingApprovals]);

  const handleHeaderApprove = async (id: number) => {
    try {
      await dailyLogsAPI.approve(id);
      window.dispatchEvent(new Event('transactionsUpdated'));
    } catch {
      // ignore
    }
  };

  const handleHeaderReject = async (id: number) => {
    try {
      await dailyLogsAPI.reject(id);
      loadPendingApprovals();
    } catch {
      // ignore
    }
  };

  const needsName = !user?.full_name && !user?.must_set_password;
  const mustSetPassword = Boolean(user?.must_set_password);

  useEffect(() => {
    if (user?.full_name) {
      setForcedFullName(user.full_name);
    }
  }, [user?.full_name]);

  useEffect(() => {
    const standalone =
      (window.matchMedia && window.matchMedia('(display-mode: standalone)').matches) ||
      (window.navigator as any).standalone === true;
    setIsStandalonePWA(standalone);

    const onBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setDeferredInstallPrompt(e);
    };
    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt);
    return () => window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt);
  }, []);

  const handleInstallAppClick = async () => {
    if (deferredInstallPrompt) {
      deferredInstallPrompt.prompt();
      try {
        await deferredInstallPrompt.userChoice;
      } catch {
        // ignore
      }
      setDeferredInstallPrompt(null);
    } else {
      setShowInstallGuide(true);
    }
  };

  const handleMandatoryPasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setForcedError('');
    if (!forcedPassword.trim() || forcedPassword.trim().length < 6) {
      setForcedError('Password must be at least 6 characters');
      return;
    }
    if (forcedPassword.trim() !== forcedConfirmPassword.trim()) {
      setForcedError('Passwords do not match');
      return;
    }

    setSaving(true);
    try {
      if (forcedFullName.trim() && forcedFullName.trim() !== user?.full_name) {
        await usersAPI.updateProfile({ full_name: forcedFullName.trim() });
      }
      await usersAPI.changePassword({
        new_password: forcedPassword.trim(),
        logout_other_devices: false,
      });
      updateUser({
        must_set_password: false,
        ...(forcedFullName.trim() ? { full_name: forcedFullName.trim() } : {}),
      });
      setForcedPassword('');
      setForcedConfirmPassword('');
    } catch (err: any) {
      setForcedError(err.response?.data?.error || 'Failed to save password');
    } finally {
      setSaving(false);
    }
  };

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
          {!isStandalonePWA && (
            <button
              type="button"
              className="pwa-install-btn"
              data-tour="install"
              onClick={handleInstallAppClick}
              title={t('pwa.installTitle')}
            >
              {t('pwa.installApp')}
            </button>
          )}
          {canApproveDailyLogs && (
            <div style={{ position: 'relative' }}>
              <button
                type="button"
                aria-label="Pending approval notifications"
                onClick={() => setShowNotifications((prev) => !prev)}
                style={{
                  position: 'relative',
                  background: '#f1f5f9',
                  border: '1px solid #cbd5e1',
                  borderRadius: '999px',
                  padding: '0.35rem 0.65rem',
                  cursor: 'pointer',
                  fontSize: '0.95rem',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.3rem',
                }}
              >
                <span>🔔</span>
                {pendingApprovals.length > 0 && (
                  <span
                    data-testid="notification-badge"
                    style={{
                      background: '#dc2626',
                      color: '#fff',
                      borderRadius: '999px',
                      fontSize: '0.72rem',
                      fontWeight: 700,
                      padding: '0.05rem 0.4rem',
                      lineHeight: 1.3,
                    }}
                  >
                    {pendingApprovals.length}
                  </span>
                )}
              </button>

              {showNotifications && (
                <div
                  style={{
                    position: 'absolute',
                    right: 0,
                    top: 'calc(100% + 8px)',
                    width: '320px',
                    maxWidth: '90vw',
                    background: '#fff',
                    border: '1px solid #e2e8f0',
                    borderRadius: '10px',
                    boxShadow: '0 10px 25px rgba(0,0,0,0.12)',
                    padding: '0.75rem',
                    zIndex: 1000,
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                    <strong style={{ fontSize: '0.9rem' }}>Pending Approvals ({pendingApprovals.length})</strong>
                    <button
                      type="button"
                      onClick={() => setShowNotifications(false)}
                      style={{ border: 'none', background: 'transparent', cursor: 'pointer', fontSize: '1rem' }}
                    >
                      ✕
                    </button>
                  </div>
                  {pendingApprovals.length === 0 ? (
                    <p style={{ margin: '0.5rem 0', fontSize: '0.85rem', color: '#64748b' }}>
                      No pending employee submissions.
                    </p>
                  ) : (
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', maxHeight: '280px', overflowY: 'auto' }}>
                      {pendingApprovals.map((item) => (
                        <div
                          key={item.id}
                          style={{
                            padding: '0.55rem',
                            border: '1px solid #e2e8f0',
                            borderRadius: '8px',
                            fontSize: '0.82rem',
                          }}
                        >
                          <div style={{ fontWeight: 600 }}>
                            {item.log_type.replace('_', ' ')} · {item.log_date}
                          </div>
                          <div style={{ color: '#334155', margin: '0.15rem 0' }}>
                            {item.item_name || item.category}
                            {item.quantity > 0 ? ` (${item.quantity} ${item.unit})` : ''}
                            {item.amount > 0 ? ` — ₹${item.amount.toLocaleString('en-IN')}` : ''}
                          </div>
                          <div style={{ color: '#64748b', fontSize: '0.76rem', marginBottom: '0.4rem' }}>
                            By {item.submitted_by_name || 'Employee'}
                          </div>
                          <div style={{ display: 'flex', gap: '0.4rem' }}>
                            <button
                              type="button"
                              onClick={() => handleHeaderApprove(item.id)}
                              style={{ padding: '0.25rem 0.6rem', borderRadius: '6px', border: 'none', background: '#16a34a', color: '#fff', fontWeight: 600, cursor: 'pointer', fontSize: '0.78rem' }}
                            >
                              Approve
                            </button>
                            <button
                              type="button"
                              onClick={() => handleHeaderReject(item.id)}
                              style={{ padding: '0.25rem 0.6rem', borderRadius: '6px', border: '1px solid #fecaca', background: '#fef2f2', color: '#dc2626', fontWeight: 600, cursor: 'pointer', fontSize: '0.78rem' }}
                            >
                              Reject
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          <div className="lang-switcher" role="group" data-tour="language" aria-label={t('header.languageLabel')}>
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
                      <button
                        onClick={() => {
                          setShowProfileMenu(false);
                          window.dispatchEvent(new Event('openOnboardingTour'));
                        }}
                        className="secondary-action-btn"
                      >
                        {t('tour.replay')}
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

      {showInstallGuide && (
        <div className="header-modal-backdrop" onClick={() => setShowInstallGuide(false)}>
          <div className="header-modal-card" onClick={e => e.stopPropagation()}>
            <h3>{t('pwa.installTitle')}</h3>
            <p className="header-modal-desc">{t('pwa.installGuide')}</p>
            <button type="button" className="save-name-btn" onClick={() => setShowInstallGuide(false)}>
              OK
            </button>
          </div>
        </div>
      )}

      {mustSetPassword && (
        <div className="header-modal-backdrop mandatory-password-backdrop" data-testid="mandatory-password-modal">
          <div className="header-modal-card mandatory-password-card">
            <h3>{t('magic.setPasswordTitle')}</h3>
            <p className="header-modal-desc">{t('magic.setPasswordDesc')}</p>
            {forcedError && <div className="mandatory-password-error">{forcedError}</div>}
            <form onSubmit={handleMandatoryPasswordSubmit} className="mandatory-password-form">
              <label>
                <span>{t('magic.fullNameLabel')}</span>
                <input
                  type="text"
                  value={forcedFullName}
                  onChange={e => setForcedFullName(e.target.value)}
                  placeholder="Enter your name"
                  className="password-input"
                  disabled={saving}
                />
              </label>
              <label>
                <span>{t('magic.newPasswordLabel')}</span>
                <input
                  type="password"
                  value={forcedPassword}
                  onChange={e => setForcedPassword(e.target.value)}
                  placeholder="Minimum 6 characters"
                  className="password-input"
                  minLength={6}
                  required
                  autoFocus
                  disabled={saving}
                />
              </label>
              <label>
                <span>{t('magic.confirmPasswordLabel')}</span>
                <input
                  type="password"
                  value={forcedConfirmPassword}
                  onChange={e => setForcedConfirmPassword(e.target.value)}
                  placeholder="Re-enter password"
                  className="password-input"
                  minLength={6}
                  required
                  disabled={saving}
                />
              </label>
              <button type="submit" className="mandatory-save-btn" disabled={saving}>
                {saving ? t('header.saving') : t('magic.savePasswordBtn')}
              </button>
            </form>
          </div>
        </div>
      )}
    </>
  );
};

export default Header;

