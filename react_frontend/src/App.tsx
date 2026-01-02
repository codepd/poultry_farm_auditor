import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import ProtectedRoute from './components/ProtectedRoute';
import Layout from './components/Layout/Layout';
import LoginPage from './pages/LoginPage';
import AcceptInvitePage from './pages/AcceptInvitePage';
import HomePage from './pages/HomePage';
import ExpensesPage from './pages/ExpensesPage';
import HenBatchesPage from './pages/HenBatchesPage';
import PriceHistoryPage from './pages/PriceHistoryPage';
import './App.css';

function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/accept-invite" element={<AcceptInvitePage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Layout>
                  <HomePage />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/expenses"
            element={
              <ProtectedRoute>
                <Layout>
                  <ExpensesPage />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/hen-batches"
            element={
              <ProtectedRoute>
                <Layout>
                  <HenBatchesPage />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route
            path="/price-history"
            element={
              <ProtectedRoute>
                <Layout>
                  <PriceHistoryPage />
                </Layout>
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;
