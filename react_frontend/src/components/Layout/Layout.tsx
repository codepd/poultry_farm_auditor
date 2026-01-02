import React from 'react';
import Header from './Header';
import Navbar from './Navbar';
import './Layout.css';

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  return (
    <div className="app-layout">
      <Header />
      <Navbar />
      <main className="app-main">
        {children}
      </main>
    </div>
  );
};

export default Layout;




