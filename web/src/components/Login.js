import React, { useState, useContext } from 'react';
import API from '../api/api';
import { Link, useNavigate } from 'react-router-dom';
import { WebSocketContext } from '../WebsocketContext';
import '../App.css';

function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const navigate = useNavigate();
  const { setToken } = useContext(WebSocketContext);

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await API.post('/api/auth/login', { username, password });
      if (response.data.error) {
        setErrorMessage(response.data.message || 'Login failed, please try again.');
        return;
      }
      const jwt = response.data.token;
      localStorage.setItem('sc2-token', jwt);
      setToken(jwt);
      navigate('/dashboard');
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

        <div className="auth-card-title">LOGIN</div>

        <form onSubmit={handleSubmit} className="auth-form">
          {errorMessage && <div className="auth-error">{errorMessage}</div>}

          <div className="auth-field">
            <label className="auth-label">USERNAME</label>
            <input
              type="text"
              className="auth-input"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder="Enter username"
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
              placeholder="Enter password"
            />
          </div>

          <button type="submit" className="auth-submit">AUTHENTICATE</button>
        </form>

        <div className="auth-footer">
          <span className="auth-footer-text">
            No account?<Link to="/signup" className="auth-footer-link">CREATE ONE</Link>
          </span>
        </div>
      </div>
    </div>
  );
}

export default Login;
