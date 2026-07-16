import { api, priorityLabel, repeatLabel, statusLabel } from '../api';
import type { ScheduleItem } from '../types';

export function renderItemCard(item: ScheduleItem, onClick: (item: ScheduleItem) => void): HTMLElement {
  const el = document.createElement('div');
  el.className = `card item-card priority-${item.priority}${item.status === 'completed' ? ' completed' : ''}`;
  el.innerHTML = `
    <div class="title">${escapeHtml(item.title)}</div>
    <div class="meta">
      ${item.hasTime ? `<span>${item.startTime}${item.endTime && item.endTime !== '00:00' ? ' - ' + item.endTime : ''}</span>` : '<span>全天</span>'}
      <span class="tag">${statusLabel(item.status)}</span>
      <span class="tag">${priorityLabel(item.priority)}</span>
      ${item.repeat !== 'none' ? `<span class="tag">${repeatLabel(item.repeat)}</span>` : ''}
      ${item.category ? `<span class="tag">${escapeHtml(item.category)}</span>` : ''}
    </div>
    <div class="item-quick-actions">
      ${item.executionStatus === 'executed' ? '<span class="execution-done">已完成</span>' : item.executionStatus === 'skipped' ? '<span class="execution-skipped">未完成</span>' : '<button type="button" data-execution="executed">已完成</button><button type="button" data-execution="skipped">未完成</button>'}
      <button type="button" class="item-edit-action" data-edit>编辑</button>
    </div>
  `;
  el.addEventListener('click', () => onClick(item));
  el.querySelectorAll<HTMLButtonElement>('[data-execution]').forEach((button) => {
    button.addEventListener('click', async (event) => {
      event.stopPropagation();
      button.disabled = true;
      try {
        await api.setExecution(item.id, button.dataset.execution ?? 'not_started');
        window.dispatchEvent(new CustomEvent('chrona-schedule-updated'));
      } finally {
        button.disabled = false;
      }
    });
  });
  el.querySelector<HTMLButtonElement>('[data-edit]')?.addEventListener('click', (event) => {
    event.stopPropagation();
    onClick(item);
  });
  return el;
}

export function escapeHtml(text: string): string {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

export function renderEmpty(message = '暂无计划'): HTMLElement {
  const el = document.createElement('div');
  el.className = 'empty-state';
  el.textContent = message;
  return el;
}
