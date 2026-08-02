import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from './App';
import './styles/index.css';

const deploymentRecoveryKey = "signalops.deployment-recovery";

function recoverFromStaleDeployment(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  if (!message.includes("dynamically imported module") && !message.includes("Failed to fetch dynamically imported module")) return;
  if (sessionStorage.getItem(deploymentRecoveryKey) === location.href) return;
  sessionStorage.setItem(deploymentRecoveryKey, location.href);
  location.reload();
}

window.addEventListener("vite:preloadError", (event) => {
  event.preventDefault();
  recoverFromStaleDeployment(new Error("Failed to fetch dynamically imported module"));
});
window.addEventListener("error", (event) => recoverFromStaleDeployment(event.error ?? event.message), true);
window.addEventListener("unhandledrejection", (event) => recoverFromStaleDeployment(event.reason));

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
