import React from 'react';
import { useAuth } from '../../context/AuthContext';
import './Header.css';

const Header: React.FC = () => {
  const { currentTenant, user, logout } = useAuth();
  const [showProfileMenu, setShowProfileMenu] = React.useState(false);

  const getInitials = (name?: string) => {
    if (!name) return 'U';
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  // Close dropdown when clicking outside
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest('.profile-menu')) {
        setShowProfileMenu(false);
      }
    };

    if (showProfileMenu) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [showProfileMenu]);

  return (
    <header className="app-header">
      <div className="header-left">
        <h1 className="farm-name">{currentTenant?.name || 'Farm'}</h1>
      </div>
      <div className="header-right">
        <div className="profile-menu" onClick={() => setShowProfileMenu(!showProfileMenu)}>
          <div className="profile-icon">
            {user?.full_name ? getInitials(user.full_name) : 'U'}
          </div>
          {showProfileMenu && (
            <div className="profile-dropdown">
              <div className="profile-info">
                <div className="profile-name">{user?.full_name || 'User'}</div>
                <div className="profile-email">{user?.email}</div>
                {currentTenant && (
                  <div className="profile-role">
                    <span className="role-badge">{currentTenant.role}</span>
                  </div>
                )}
              </div>
              <div className="profile-actions">
                <button onClick={logout} className="logout-btn">
                  Logout
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};

export default Header;

