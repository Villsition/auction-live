import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';

// Minimal CSS reset
const style = document.createElement('style');
style.textContent = `
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #2d3748; background: #f7fafc; }
  a { text-decoration: none; }
  input, select, button { font-family: inherit; outline: none; }
  button:active { transform: scale(0.97); }
`;
document.head.appendChild(style);

createRoot(document.getElementById('root')!).render(
  <StrictMode><App /></StrictMode>
);
