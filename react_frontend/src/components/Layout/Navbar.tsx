import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import './Navbar.css';

const Navbar: React.FC = () => {
  const location = useLocation();

  const navItems = [
    { path: '/', label: 'Home', icon: '🏠' },
    { path: '/expenses', label: 'Expenses', icon: '💰' },
    { path: '/hen-batches', label: 'Hen Batches', icon: '🐔' },
    { path: '/price-history', label: 'Price History', icon: '📈' },
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

