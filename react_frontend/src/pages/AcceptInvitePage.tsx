import React, { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import api from '../services/api';
import './AcceptInvitePage.css';

const AcceptInvitePage: React.FC = () => {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token');
  const navigate = useNavigate();
  
  const [fullName, setFullName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [inviteInfo, setInviteInfo] = useState<{ email: string; role: string } | null>(null);

  useEffect(() => {
    if (!token) {
      setError('Invalid invitation link. No token provided.');
      return;
    }

    // Optionally verify token and get invite info
    // For now, we'll just show the form
  }, [token]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!token) {
      setError('Invalid invitation link');
      return;
    }

    if (password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (!fullName.trim()) {
      setError('Full name is required');
      return;
    }

    setLoading(true);

    try {
      await api.post('/users/accept-invite', {
        token,
        password,
        full_name: fullName,
      });

      alert('Account activated successfully! Please login.');
      navigate('/login');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to accept invitation');
    } finally {
      setLoading(false);
    }
  };

  if (!token) {
    return (
      <div className="accept-invite-page">
        <div className="accept-invite-container">
          <h1>Invalid Invitation</h1>
          <p>The invitation link is invalid or has expired.</p>
          <button onClick={() => navigate('/login')}>Go to Login</button>
        </div>
      </div>
    );
  }

  return (
    <div className="accept-invite-page">
      <div className="accept-invite-container">
        <h1>Poultry Farm</h1>
        <h2>Set Up Your Account</h2>
        {inviteInfo && (
          <div className="invite-info">
            <p>Email: {inviteInfo.email}</p>
            <p>Role: {inviteInfo.role}</p>
          </div>
        )}
        <form onSubmit={handleSubmit}>
          {error && <div className="error-message">{error}</div>}
          <div className="form-group">
            <label>Full Name</label>
            <input
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              required
              disabled={loading}
              placeholder="Enter your full name"
            />
          </div>
          <div className="form-group">
            <label>Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={loading}
              placeholder="Minimum 6 characters"
              minLength={6}
            />
          </div>
          <div className="form-group">
            <label>Confirm Password</label>
            <input
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              disabled={loading}
              placeholder="Re-enter your password"
              minLength={6}
            />
          </div>
          <button type="submit" disabled={loading}>
            {loading ? 'Setting up...' : 'Activate Account'}
          </button>
        </form>
      </div>
    </div>
  );
};

export default AcceptInvitePage;


