import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { useLanguage } from '../../context/LanguageContext';
import './Navbar.css';

const MANAGE_ROLES = ['OWNER', 'ADMIN', 'CO_OWNER'];
const EDIT_ROLES = ['OWNER', 'ADMIN', 'CO_OWNER', 'MANAGER'];

const Navbar: React.FC = () => {
  const location = useLocation();
  const { currentTenant } = useAuth();
  const { t } = useLanguage();

  const canManageUsers = Boolean(currentTenant && MANAGE_ROLES.includes(currentTenant.role));
  const canEdit = Boolean(currentTenant && EDIT_ROLES.includes(currentTenant.role));

  const navItems = [
    { path: '/', label: t('nav.home'), mobileLabel: t('nav.home'), icon: '🏠' },
    ...(canEdit ? [{ path: '/expenses', label: t('nav.expenses'), mobileLabel: t('nav.expenses'), icon: '💰' }] : []),
    { path: '/hen-batches', label: t('nav.henBatches'), mobileLabel: t('nav.henBatchesShort'), icon: '🐔' },
    { path: '/price-history', label: t('nav.priceHistory'), mobileLabel: t('nav.priceHistoryShort'), icon: '📈' },
    ...(canEdit ? [{ path: '/upload-sheets', label: t('nav.uploadSheets'), mobileLabel: t('nav.uploadSheetsShort'), icon: '📤' }] : []),
    ...(canManageUsers ? [{ path: '/team', label: t('nav.team'), mobileLabel: t('nav.team'), icon: '👥' }] : []),
  ];

  return (
    <nav className="app-navbar" data-tour="nav">
      <ul className="nav-list">
        {navItems.map((item) => (
          <li key={item.path}>
            <Link
              to={item.path}
              className={`nav-link ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              <span className="nav-label nav-label-desktop">{item.label}</span>
              <span className="nav-label nav-label-mobile">{item.mobileLabel}</span>
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
};

export default Navbar;

