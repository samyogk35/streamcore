import React from 'react';
import { Link } from 'react-router-dom';
import '../App.css';

function LandingPage() {
  return (
    <div className="landing-container">
      <div className="chat-window">
        <div className="login-panel">
          <div className='headings'>StreamCore</div>
          <span>Real-time market data — distributed, low-latency, built in Go</span>
          <br />
          <br />
          <div className="form-button">
            <Link to="/signup"><button className="landing-button">Create a new Account</button></Link>
            <Link to="/login"><button className="landing-button">Already a user? Login here</button></Link>
          </div>
        </div>
      </div>
    </div>
  );
}

export default LandingPage;
