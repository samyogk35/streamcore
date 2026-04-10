import React, { useState } from 'react';
import API from '../api/api';
import { Link, useNavigate } from 'react-router-dom';
import '../App.css';

function Signup() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const navigate = useNavigate();

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await API.post('/api/auth/signup', { username, password });
      if (response.data.error) {
        setErrorMessage(response.data.message || 'Signup failed. Please try again.');
        return;
      }
      navigate('/login');
    } catch (error) {
      setErrorMessage('Connection failed. Check your network.');
    }
  };

  return (
    <div className="sc-auth-page">
      <div className="sc-auth-card">
        <div className="auth-card-header">
          <div className="auth-brand">
            <span className="auth-brand-text">STREAMCORE</span>
            <span className="auth-brand-ver">v2</span>
          </div>
          <div className="auth-subtitle">REAL-TIME MARKET DATA PLATFORM</div>
        </div>

        <div className="auth-card-title">CREATE ACCOUNT</div>

        <form onSubmit={handleSubmit} className="auth-form">
          {errorMessage && <div className="auth-error">{errorMessage}</div>}

          <div className="auth-field">
            <label className="auth-label">USERNAME</label>
            <input
              type="text"
              className="auth-input"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="Choose a username"
              autoFocus
            />
          </div>

          <div className="auth-field">
            <label className="auth-label">PASSWORD</label>
            <input
              type="password"
              className="auth-input"
              value={password}
              onChange={e => setPassword(e.target.value)}
              placeholder="Create a strong password"
            />
          </div>

          <button type="submit" className="auth-submit">REGISTER</button>
        </form>

        <div className="auth-footer">
          <span className="auth-footer-text">
            Already registered?<Link to="/login" className="auth-footer-link">LOGIN</Link>
          </span>
        </div>
      </div>
    </div>
  );
}

export default Signup;
