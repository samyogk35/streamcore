import './App.css';
import React from 'react';
import Signup from './components/Signup';
import Login from './components/Login';
import LandingPage from './components/LandingPage';
import Dashboard from './components/Dashboard';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { WebSocketProvider } from './WebsocketContext';

function ProtectedRoute({ children }) {
  const token = localStorage.getItem('sc2-token');
  return token ? children : <Navigate to="/login" replace />;
}

function App() {
  return (
    <WebSocketProvider>
      <Router>
        <Routes>
          <Route
            path="/"
            element={
              localStorage.getItem('sc2-token')
                ? <Navigate to="/dashboard" replace />
                : <LandingPage />
            }
          />
          <Route path="/signup" element={<Signup />} />
          <Route path="/login" element={<Login />} />
          <Route
            path="/dashboard"
            element={<ProtectedRoute><Dashboard /></ProtectedRoute>}
          />
        </Routes>
      </Router>
    </WebSocketProvider>
  );
}

export default App;
