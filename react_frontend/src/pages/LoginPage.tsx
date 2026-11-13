import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { authAPI, CountryCodeInfo } from '../services/api';
import './LoginPage.css';

type LoginMode = 'email' | 'phone';
type PhoneStep = 'phone-password' | 'enter-phone' | 'enter-otp';

const LoginPage: React.FC = () => {
  const [mode, setMode] = useState<LoginMode>('phone');
  const { login, loginWithOTP, loginWithMagicLink } = useAuth();
  const navigate = useNavigate();

  // Email login state
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  // Phone login state
  const [phoneStep, setPhoneStep] = useState<PhoneStep>('phone-password');
  const [countryCodes, setCountryCodes] = useState<CountryCodeInfo[]>([]);
  const [selectedCountryCode, setSelectedCountryCode] = useState('+91');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [phonePassword, setPhonePassword] = useState('');
  const [otpCode, setOtpCode] = useState('');
  const [otpCountdown, setOtpCountdown] = useState(0);
  const [rememberMe, setRememberMe] = useState(true);

  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // WhatsApp Magic Link Verification & Password Setup state (?magic_token=...)
  // Does NOT auto-consume on GET/load (prevents WhatsApp preview bots & link forwarding attacks).
  const [magicToken, setMagicToken] = useState<string | null>(() => {
    const params = new URLSearchParams(window.location.search);
    return params.get('magic_token');
  });
  const [magicFullName, setMagicFullName] = useState('');
  const [magicNewPassword, setMagicNewPassword] = useState('');
  const [magicConfirmPassword, setMagicConfirmPassword] = useState('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tok = params.get('magic_token');
    if (tok) setMagicToken(tok);
  }, []);

  const handleMagicLinkSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!magicToken) return;
    if (!phoneNumber.trim()) {
      setError('Please enter your invited phone number to verify your identity');
      return;
    }
    if (!magicNewPassword.trim() || magicNewPassword.trim().length < 6) {
      setError('Password / PIN must be at least 6 characters');
      return;
    }
    if (magicNewPassword.trim() !== magicConfirmPassword.trim()) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);
    try {
      await loginWithMagicLink(magicToken, rememberMe, {
        phone: fullPhone,
        full_name: magicFullName.trim() || undefined,
        password: magicNewPassword.trim(),
      });
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Verification failed. Ensure your phone number matches the invitation.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    authAPI.getCountryCodes().then(codes => {
      if (codes.length > 0) {
        setCountryCodes(codes);
        setSelectedCountryCode(codes[0].country_code);
      }
    }).catch(() => {});
  }, []);

  useEffect(() => {
    if (otpCountdown <= 0) return;
    const timer = setTimeout(() => setOtpCountdown(c => c - 1), 1000);
    return () => clearTimeout(timer);
  }, [otpCountdown]);

  const fullPhone = `${selectedCountryCode}${phoneNumber}`;

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await login(email, password, rememberMe);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSendOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!phoneNumber.trim()) {
      setError('Please enter your phone number');
      return;
    }
    setLoading(true);
    try {
      await authAPI.sendOTP({ phone: fullPhone, country_code: selectedCountryCode });
      setPhoneStep('enter-otp');
      setOtpCountdown(300);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to send OTP');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!otpCode.trim()) {
      setError('Please enter the OTP');
      return;
    }
    setLoading(true);
    try {
      await loginWithOTP(fullPhone, otpCode, rememberMe);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Invalid OTP');
    } finally {
      setLoading(false);
    }
  };

  const handleResendOTP = async () => {
    setError('');
    setLoading(true);
    try {
      await authAPI.sendOTP({ phone: fullPhone, country_code: selectedCountryCode });
      setOtpCountdown(300);
      setOtpCode('');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to resend OTP');
    } finally {
      setLoading(false);
    }
  };

  const handlePhonePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!phoneNumber.trim() || !phonePassword.trim()) {
      setError('Please enter your phone number and password');
      return;
    }
    setLoading(true);
    try {
      await login(fullPhone, phonePassword, rememberMe);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  const switchToPhone = () => {
    setMode('phone');
    setError('');
    setPhoneStep('phone-password');
    setOtpCode('');
  };

  const switchToEmail = () => {
    setMode('email');
    setError('');
  };

  const formatCountdown = (seconds: number) => {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  };

  return (
    <div className="login-page">
      <div className="login-container">
        <h1>Poultry Farm</h1>
        <h2>{magicToken ? 'Verify Phone & Set Password' : 'Sign In'}</h2>

        {!magicToken && (
          <div className="login-tabs">
            <button
              type="button"
              className={`login-tab ${mode === 'phone' ? 'active' : ''}`}
              onClick={switchToPhone}
            >
              Phone
            </button>
            <button
              type="button"
              className={`login-tab ${mode === 'email' ? 'active' : ''}`}
              onClick={switchToEmail}
            >
              Email
            </button>
          </div>
        )}

        {error && <div className="error-message">{error}</div>}

        {magicToken && (
          <form onSubmit={handleMagicLinkSubmit} data-testid="magic-link-verify-form">
            <p style={{ fontSize: '0.86rem', color: '#475569', marginBottom: '1rem', lineHeight: 1.45 }}>
              For your security, please verify the phone number this WhatsApp link was sent to and create your password/PIN.
            </p>
            <div className="form-group">
              <label>Verify Your Phone Number</label>
              <div className="phone-input-row">
                <select
                  className="country-code-select"
                  value={selectedCountryCode}
                  onChange={(e) => setSelectedCountryCode(e.target.value)}
                  disabled={loading}
                  aria-label="Country code"
                >
                  {countryCodes.length > 0 ? (
                    countryCodes.map(cc => (
                      <option key={cc.country_code} value={cc.country_code}>
                        {cc.country_code}
                      </option>
                    ))
                  ) : (
                    <>
                      <option value="+91">+91</option>
                      <option value="+1">+1</option>
                    </>
                  )}
                </select>
                <input
                  type="tel"
                  className="phone-number-input"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value.replace(/\D/g, ''))}
                  placeholder="9876543210"
                  disabled={loading}
                  required
                />
              </div>
            </div>
            <div className="form-group">
              <label>Your Full Name</label>
              <input
                type="text"
                value={magicFullName}
                onChange={(e) => setMagicFullName(e.target.value)}
                placeholder="Enter your full name"
                disabled={loading}
              />
            </div>
            <div className="form-group">
              <label>Set Password / PIN (min 6 characters)</label>
              <input
                type="password"
                value={magicNewPassword}
                onChange={(e) => setMagicNewPassword(e.target.value)}
                placeholder="Create a password or 6-digit PIN"
                minLength={6}
                disabled={loading}
                required
              />
            </div>
            <div className="form-group">
              <label>Confirm Password / PIN</label>
              <input
                type="password"
                value={magicConfirmPassword}
                onChange={(e) => setMagicConfirmPassword(e.target.value)}
                placeholder="Confirm your password or PIN"
                minLength={6}
                disabled={loading}
                required
              />
            </div>
            <button type="submit" disabled={loading}>
              {loading ? 'Verifying & Signing in...' : 'Verify Phone & Set Password'}
            </button>
          </form>
        )}

        {!magicToken && mode === 'email' && (
          <form onSubmit={handleEmailSubmit}>
            <div className="form-group">
              <label>Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                disabled={loading}
                placeholder="you@example.com"
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
                placeholder="Enter password"
              />
            </div>
            <label className="remember-me-row">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={loading}
              />
              <span>Remember me</span>
            </label>
            <button type="submit" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
        )}

        {!magicToken && mode === 'phone' && phoneStep === 'phone-password' && (
          <form onSubmit={handlePhonePasswordSubmit}>
            <div className="form-group">
              <label>Phone Number</label>
              <div className="phone-input-row">
                <select
                  className="country-code-select"
                  value={selectedCountryCode}
                  onChange={(e) => setSelectedCountryCode(e.target.value)}
                  disabled={loading}
                  aria-label="Country code"
                >
                  {countryCodes.length > 0 ? (
                    countryCodes.map(cc => (
                      <option key={cc.country_code} value={cc.country_code}>
                        {cc.country_code}
                      </option>
                    ))
                  ) : (
                    <>
                      <option value="+91">+91</option>
                      <option value="+1">+1</option>
                    </>
                  )}
                </select>
                <input
                  type="tel"
                  className="phone-number-input"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value.replace(/\D/g, ''))}
                  placeholder="9876543210"
                  disabled={loading}
                  required
                />
              </div>
            </div>
            <div className="form-group">
              <label>Password</label>
              <input
                type="password"
                value={phonePassword}
                onChange={(e) => setPhonePassword(e.target.value)}
                placeholder="Enter your password"
                disabled={loading}
                required
              />
            </div>
            <label className="remember-me-row">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={loading}
              />
              <span>Remember me</span>
            </label>
            <button type="submit" disabled={loading || !phoneNumber.trim() || !phonePassword.trim()}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
            <div className="resend-row">
              <button
                type="button"
                className="resend-link"
                onClick={() => { setPhoneStep('enter-phone'); setError(''); }}
                disabled={loading}
              >
                Use 6-Digit OTP instead
              </button>
            </div>
          </form>
        )}

        {!magicToken && mode === 'phone' && phoneStep === 'enter-phone' && (
          <form onSubmit={handleSendOTP}>
            <div className="form-group">
              <label>Phone Number</label>
              <div className="phone-input-row">
                <select
                  className="country-code-select"
                  value={selectedCountryCode}
                  onChange={(e) => setSelectedCountryCode(e.target.value)}
                  disabled={loading}
                  aria-label="Country code"
                >
                  {countryCodes.length > 0 ? (
                    countryCodes.map(cc => (
                      <option key={cc.country_code} value={cc.country_code}>
                        {cc.country_code}
                      </option>
                    ))
                  ) : (
                    <>
                      <option value="+91">+91</option>
                      <option value="+1">+1</option>
                    </>
                  )}
                </select>
                <input
                  type="tel"
                  className="phone-number-input"
                  value={phoneNumber}
                  onChange={(e) => setPhoneNumber(e.target.value.replace(/\D/g, ''))}
                  placeholder="9876543210"
                  disabled={loading}
                  autoFocus
                />
              </div>
            </div>
            <button type="submit" disabled={loading || !phoneNumber.trim()}>
              {loading ? 'Sending OTP...' : 'Send OTP'}
            </button>
            <div className="resend-row">
              <button
                type="button"
                className="resend-link"
                onClick={() => { setPhoneStep('phone-password'); setError(''); }}
                disabled={loading}
              >
                Sign in with Phone + Password
              </button>
            </div>
          </form>
        )}

        {!magicToken && mode === 'phone' && phoneStep === 'enter-otp' && (
          <form onSubmit={handleVerifyOTP}>
            <div className="otp-sent-info">
              OTP sent to <strong>{fullPhone}</strong>
              <button
                type="button"
                className="change-phone-link"
                onClick={() => { setPhoneStep('enter-phone'); setOtpCode(''); setError(''); }}
              >
                Change
              </button>
            </div>
            <div className="form-group">
              <label>Enter OTP</label>
              <input
                type="text"
                className="otp-input"
                value={otpCode}
                onChange={(e) => setOtpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="000000"
                maxLength={6}
                disabled={loading}
                autoFocus
                autoComplete="one-time-code"
              />
            </div>
            <label className="remember-me-row">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={loading}
              />
              <span>Remember me</span>
            </label>
            <button type="submit" disabled={loading || otpCode.length < 6}>
              {loading ? 'Verifying...' : 'Verify & Sign In'}
            </button>
            <div className="resend-row">
              {otpCountdown > 0 ? (
                <span className="resend-timer">Resend in {formatCountdown(otpCountdown)}</span>
              ) : (
                <button
                  type="button"
                  className="resend-link"
                  onClick={handleResendOTP}
                  disabled={loading}
                >
                  Resend OTP
                </button>
              )}
            </div>
          </form>
        )}
      </div>
    </div>
  );
};

export default LoginPage;
