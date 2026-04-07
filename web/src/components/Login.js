import React, { useState, useContext } from 'react';
import API from '../api/api';
import HomeButton from './HomeButton';
import { useNavigate } from 'react-router-dom';
import { WebSocketContext } from '../WebsocketContext';

function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const navigate = useNavigate();
  const { setToken } = useContext(WebSocketContext);

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
        const response = await API.post('/api/auth/login', {
            username,
            password
        });
        if (response.data.error) {
          console.log(response.data)
          setErrorMessage(response.data.message || 'Login failed, please try again.');
          return;
        }

        const jwt = response.data.token;
        localStorage.setItem('sc2-token', jwt);
        setToken(jwt);

        navigate('/dashboard');
    } catch (error) {
      console.error('Login failed', error);
    }
  };

  return (
    <div className="bg-container">
      <div className="chat-window">
        <HomeButton />
        <form onSubmit={handleSubmit} className='form'>
          <div className='headings'>Login</div>
          {errorMessage && <span className="error-message">*{errorMessage}</span>}
          <br />
          <br />
          <div className='form-input-box'>
            <input className='form-input' type="text" value={username} onChange={e => setUsername(e.target.value)} placeholder="Username" />
            <input className='form-input' type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Password" />
          </div>
          <button type="submit" className='form-button'>Login</button>
        </form>
      </div>
    </div>
  );
}

export default Login;
