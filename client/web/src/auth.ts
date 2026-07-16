import { api } from './api';

export function renderAuth(root: HTMLElement, onReady: () => void): void {
  let mode: 'login' | 'register' = 'login';
  let busy = false;

  const paint = (error = '') => {
    root.innerHTML = `
      <main class="auth-screen">
        <section class="auth-panel">
          <div class="auth-copy">
            <p class="auth-kicker">Chrona 时序</p>
            <h1>时间安排计划</h1>
            <p>登录后可在网页、手机和桌面之间同步计划。</p>
          </div>
          <form class="auth-card" id="auth-form">
            <h2>${mode === 'login' ? '登录' : '注册账号'}</h2>
            ${error ? `<div class="auth-error">${escapeHtml(error)}</div>` : ''}
            <label>
              用户名
              <input id="auth-username" type="text" autocomplete="username" required maxlength="32" />
            </label>
            <label>
              密码
              <input id="auth-password" type="password" autocomplete="${mode === 'login' ? 'current-password' : 'new-password'}" minlength="6" required />
            </label>
            <button class="btn btn-accent" id="auth-submit" type="submit">${busy ? '处理中...' : mode === 'login' ? '登录' : '创建账号'}</button>
            <button class="btn" id="auth-switch" type="button">${mode === 'login' ? '没有账号？注册' : '已有账号？登录'}</button>
          </form>
        </section>
      </main>
    `;

    root.querySelector('#auth-switch')?.addEventListener('click', () => {
      mode = mode === 'login' ? 'register' : 'login';
      paint();
    });

    root.querySelector('#auth-form')?.addEventListener('submit', async (event) => {
      event.preventDefault();
      if (busy) return;
      const username = (root.querySelector('#auth-username') as HTMLInputElement).value.trim();
      const password = (root.querySelector('#auth-password') as HTMLInputElement).value;
      try {
        busy = true;
        setSubmitDisabled(root, true);
        if (mode === 'login') await api.login(username, password);
        else await api.register(username, password);
        onReady();
      } catch (err) {
        busy = false;
        paint(err instanceof Error ? err.message : String(err));
      }
    });

    (root.querySelector('#auth-username') as HTMLInputElement | null)?.focus();
  };

  paint();
}

function setSubmitDisabled(root: HTMLElement, disabled: boolean): void {
  const button = root.querySelector('#auth-submit') as HTMLButtonElement | null;
  if (!button) return;
  button.disabled = disabled;
  button.textContent = disabled ? '处理中...' : button.textContent;
}

function escapeHtml(text: string): string {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
