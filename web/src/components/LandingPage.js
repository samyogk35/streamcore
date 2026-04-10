import React from 'react';
import { Link } from 'react-router-dom';
import '../App.css';

function LandingPage() {
  return (
    <div className="sc-landing">
      <div className="landing-hero">
        <div className="landing-brand">
          STREAMCORE<span className="landing-ver">v2</span>
        </div>
        <p className="landing-tagline">
          Real-time market data streaming.<br />
          <span className="landing-accent">Distributed</span> ingestion.{' '}
          <span className="landing-accent">Low-latency</span> delivery.<br />
          Built in Go.
        </p>
      </div>

      <div className="landing-actions">
        <Link to="/signup" className="landing-btn landing-btn-primary">CREATE ACCOUNT</Link>
        <Link to="/login" className="landing-btn">LOGIN</Link>
      </div>

    </div>
  );
}

export default LandingPage;
