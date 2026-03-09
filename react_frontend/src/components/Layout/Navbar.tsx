import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import './Navbar.css';

const MANAGE_ROLES = ['OWNER', 'ADMIN', 'CO_OWNER'];

const Navbar: React.FC = () => {
  const location = useLocation();
  const { currentTenant } = useAuth();

  const canManageUsers = currentTenant && MANAGE_ROLES.includes(currentTenant.role);

  const navItems = [
    { path: '/', label: 'Home', icon: '🏠' },
    { path: '/expenses', label: 'Expenses', icon: '💰' },
    { path: '/hen-batches', label: 'Hen Batches', icon: '🐔' },
    { path: '/price-history', label: 'Price History', icon: '📈' },
    ...(canManageUsers ? [{ path: '/team', label: 'Team', icon: '👥' }] : []),
  ];

  return (
    <nav className="app-navbar">
      <ul className="nav-list">
        {navItems.map((item) => (
          <li key={item.path}>
            <Link
              to={item.path}
              className={`nav-link ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              <span className="nav-label">{item.label}</span>
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
};

export default Navbar;

