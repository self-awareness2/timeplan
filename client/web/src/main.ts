import { App } from './app';
import { api } from './api';
import { renderAuth } from './auth';
import './styles/main.css';

document.documentElement.setAttribute('data-theme', localStorage.getItem('time_planner_theme') || 'light');

const root = document.getElementById('app');
if (!root) {
  throw new Error('#app not found');
}
const appRoot = root;

async function boot(): Promise<void> {
  const app = new App(appRoot);
  await app.init();
}

function showLogin(): void {
  renderAuth(appRoot, () => {
    boot().catch((err) => {
      appRoot.innerHTML = `<div class="empty-state">启动失败: ${err}</div>`;
    });
  });
}

window.addEventListener(api.authExpiredEvent, () => {
  if (!api.isDesktop()) {
    showLogin();
  }
});

if (!api.isDesktop() && !api.hasToken()) {
  showLogin();
} else {
  boot().catch((err) => {
    if (!api.isDesktop()) {
      api.logout();
      showLogin();
      return;
    }
    appRoot.innerHTML = `<div class="empty-state">启动失败: ${err}</div>`;
  });
}
