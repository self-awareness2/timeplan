import { api } from './api';

export function renderAuth(root: HTMLElement, onReady: () => void): void {
  let mode: 'login' | 'register' = 'login';
  let busy = false;
  const paint = (error = '') => {
    root.innerHTML = `<main class="auth-screen"><section class="auth-panel"><div class="auth-copy"><div class="auth-brand"><img src="./chrona-mark.svg" alt="" /><span>Chrona 时序</span></div><h1>规划每一段重要时间</h1><p>在网页、手机和桌面之间同步你的计划。</p></div><form class="auth-card" id="auth-form"><h2>${mode === 'login' ? '登录' : '注册账号'}</h2>${error ? `<div class="auth-error">${escapeHtml(error)}</div>` : ''}<label>用户名<input id="auth-username" type="text" autocomplete="username" maxlength="32" required /></label><label>密码<input id="auth-password" type="password" autocomplete="${mode === 'login' ? 'current-password' : 'new-password'}" minlength="6" required /></label><button class="btn btn-accent" id="auth-submit" type="submit">${busy ? '处理中...' : mode === 'login' ? '登录' : '创建账号'}</button><button class="btn" id="auth-switch" type="button">${mode === 'login' ? '没有账号？注册' : '已有账号？登录'}</button></form></section></main>`;
    root.querySelector('#auth-switch')?.addEventListener('click', () => { mode = mode === 'login' ? 'register' : 'login'; paint(); });
    root.querySelector('#auth-form')?.addEventListener('submit', async (event) => {
      event.preventDefault();
      if (busy) return;
      const username = (root.querySelector('#auth-username') as HTMLInputElement).value.trim();
      const password = (root.querySelector('#auth-password') as HTMLInputElement).value;
      try { busy = true; paint(); if (mode === 'login') await api.login(username, password); else await api.register(username, password); onReady(); }
      catch (err) { busy = false; paint(err instanceof Error ? err.message : '登录失败，请稍后重试'); }
    });
    (root.querySelector('#auth-username') as HTMLInputElement | null)?.focus();
  };
  paint();
}

function escapeHtml(text: string): string { const div = document.createElement('div'); div.textContent = text; return div.innerHTML; }
