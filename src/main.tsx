import React from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';
import './refinements.css';
import './setup.css';
import './manual.css';
import { App } from './App';
import './posthog';

createRoot(document.getElementById('root')!).render(<App />);
