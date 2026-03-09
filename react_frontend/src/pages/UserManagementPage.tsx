import React, { useState, useEffect, useCallback } from 'react';
import { usersAPI, TenantUser, InvitationInfo, InviteRequest, CountryCodeInfo, authAPI } from '../services/api';
import './UserManagementPage.css';

const ROLES = [
  { value: 'ADMIN', label: 'Admin' },
  { value: 'CO_OWNER', label: 'Co-Owner' },
  { value: 'MANAGER', label: 'Manager' },
  { value: 'AUDITOR', label: 'Auditor' },
  { value: 'OTHER_USER', label: 'User' },
];

const UserManagementPage: React.FC = () => {
  const [users, setUsers] = useState<TenantUser[]>([]);
  const [invitations, setInvitations] = useState<InvitationInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  // Invite form state
  const [showInviteForm, setShowInviteForm] = useState(false);
  const [inviteMode, setInviteMode] = useState<'phone' | 'email'>('phone');
  const [inviteEmail, setInviteEmail] = useState('');
  const [invitePhone, setInvitePhone] = useState('');
  const [inviteCountryCode, setInviteCountryCode] = useState('+91');
  const [inviteRole, setInviteRole] = useState('OTHER_USER');
  const [inviting, setInviting] = useState(false);
  const [countryCodes, setCountryCodes] = useState<CountryCodeInfo[]>([]);

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const [usersData, invitationsData] = await Promise.all([
        usersAPI.getUsers(),
        usersAPI.getInvitations(),
      ]);
      setUsers(usersData);
      setInvitations(invitationsData);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
    authAPI.getCountryCodes().then(codes => {
      if (codes.length > 0) {
        setCountryCodes(codes);
        setInviteCountryCode(codes[0].country_code);
      }
    }).catch(() => {});
  }, [loadData]);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMsg('');
    setInviting(true);

    try {
      const data: InviteRequest = { role: inviteRole };
      if (inviteMode === 'phone') {
        data.phone = inviteCountryCode + invitePhone.trim();
      } else {
        data.email = inviteEmail.trim();
      }

      const result = await usersAPI.inviteUser(data);
      setSuccessMsg(result.message || 'Invitation created');
      setInviteEmail('');
      setInvitePhone('');
      setInviteRole('OTHER_USER');
      setShowInviteForm(false);
      await loadData();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to send invitation');
    } finally {
      setInviting(false);
    }
  };

  const getInviteIdentifier = (inv: InvitationInfo) => {
    if (inv.phone && inv.phone.Valid) return inv.phone.String;
    if (inv.email && inv.email.Valid) return inv.email.String;
    return '—';
  };

  const getInviteType = (inv: InvitationInfo) => {
    if (inv.phone && inv.phone.Valid) return 'Phone';
    return 'Email';
  };

  const getInviteStatus = (inv: InvitationInfo) => {
    if (inv.accepted_at) return 'accepted';
    if (new Date(inv.expires_at) < new Date()) return 'expired';
    return 'pending';
  };

  const pendingInvitations = invitations.filter(i => getInviteStatus(i) === 'pending');

  if (loading) {
    return <div className="users-page"><p className="loading-text">Loading...</p></div>;
  }

  return (
    <div className="users-page">
      <div className="page-header">
        <h1>Team</h1>
        <button className="add-btn" onClick={() => setShowInviteForm(!showInviteForm)}>
          {showInviteForm ? 'Cancel' : 'Invite Member'}
        </button>
      </div>

      {error && <div className="msg msg-error">{error}</div>}
      {successMsg && <div className="msg msg-success">{successMsg}</div>}

      {showInviteForm && (
        <div className="invite-form-card">
          <h2>Invite a new member</h2>
          <div className="invite-mode-tabs">
            <button
              type="button"
              className={`invite-tab ${inviteMode === 'phone' ? 'active' : ''}`}
              onClick={() => setInviteMode('phone')}
            >
              Phone
            </button>
            <button
              type="button"
              className={`invite-tab ${inviteMode === 'email' ? 'active' : ''}`}
              onClick={() => setInviteMode('email')}
            >
              Email
            </button>
          </div>
          <form onSubmit={handleInvite}>
            {inviteMode === 'phone' ? (
              <div className="form-group">
                <label>Phone Number</label>
                <div className="phone-row">
                  <select
                    value={inviteCountryCode}
                    onChange={e => setInviteCountryCode(e.target.value)}
                    className="country-select"
                    aria-label="Country code"
                  >
                    {countryCodes.length > 0 ? (
                      countryCodes.map(cc => (
                        <option key={cc.country_code} value={cc.country_code}>
                          {cc.country_code} {cc.country_name}
                        </option>
                      ))
                    ) : (
                      <>
                        <option value="+91">+91 India</option>
                        <option value="+1">+1 United States</option>
                      </>
                    )}
                  </select>
                  <input
                    type="tel"
                    value={invitePhone}
                    onChange={e => setInvitePhone(e.target.value.replace(/\D/g, ''))}
                    placeholder="9876543210"
                    required
                  />
                </div>
              </div>
            ) : (
              <div className="form-group">
                <label>Email</label>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={e => setInviteEmail(e.target.value)}
                  placeholder="user@example.com"
                  required
                />
              </div>
            )}
            <div className="form-group">
              <label>Role</label>
              <select value={inviteRole} onChange={e => setInviteRole(e.target.value)} aria-label="Role">
                {ROLES.map(r => (
                  <option key={r.value} value={r.value}>{r.label}</option>
                ))}
              </select>
            </div>
            <button type="submit" className="submit-btn" disabled={inviting}>
              {inviting ? 'Sending...' : 'Send Invitation'}
            </button>
          </form>
        </div>
      )}

      <section className="section">
        <h2>Members ({users.length})</h2>
        {users.length === 0 ? (
          <p className="empty-text">No members yet.</p>
        ) : (
          <div className="members-list">
            {users.map(u => (
              <div key={u.id} className="member-card">
                <div className="member-info">
                  <div className="member-name">{u.full_name || 'Unnamed'}</div>
                  <div className="member-contact">
                    {u.email && <span>{u.email}</span>}
                    {u.email && u.phone && <span className="separator"> · </span>}
                    {u.phone && <span>{u.phone}</span>}
                  </div>
                </div>
                <div className="member-meta">
                  <span className={`role-badge role-${u.role.toLowerCase()}`}>{u.role.replace('_', ' ')}</span>
                  {u.is_owner && <span className="owner-badge">Owner</span>}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {pendingInvitations.length > 0 && (
        <section className="section">
          <h2>Pending Invitations ({pendingInvitations.length})</h2>
          <div className="invitations-list">
            {pendingInvitations.map(inv => (
              <div key={inv.id} className="invitation-card">
                <div className="invitation-info">
                  <div className="invitation-to">
                    <span className="invite-type">{getInviteType(inv)}</span>
                    <span className="invite-identifier">{getInviteIdentifier(inv)}</span>
                  </div>
                  <div className="invitation-meta">
                    Expires {new Date(inv.expires_at).toLocaleDateString()}
                  </div>
                </div>
                <span className={`role-badge role-${inv.role.toLowerCase()}`}>{inv.role.replace('_', ' ')}</span>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
};

export default UserManagementPage;
