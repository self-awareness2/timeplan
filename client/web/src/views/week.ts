import { api, todayString } from '../api';
import type { ScheduleItem } from '../types';
import { escapeHtml, renderEmpty } from './common';

export async function renderWeekView(
  container: HTMLElement,
  date: string,
  onItemClick: (item: ScheduleItem) => void
): Promise<string> {
  const data = await api.listWeek(date);
  container.innerHTML = '';

  const grid = document.createElement('div');
  grid.className = 'week-grid';

  const today = todayString();

  for (const day of data.days) {
    const col = document.createElement('div');
    col.className = 'week-col';

    const header = document.createElement('div');
    header.className = 'week-col-header' + (day.date === today ? ' today' : '');
    header.innerHTML = `<div>${day.weekday}</div><div>${day.date.slice(5)}</div>`;
    col.appendChild(header);

    if (day.items.length === 0) {
      col.appendChild(renderEmpty('暂无'));
    } else {
      for (const item of day.items) {
        const el = document.createElement('div');
        el.className = 'week-item';
        el.innerHTML = `
          <div><strong>${escapeHtml(item.title)}</strong></div>
          <div style="color:var(--muted);font-size:0.75rem">
            ${item.hasTime ? item.startTime : '全天'}
          </div>
        `;
        el.addEventListener('click', () => onItemClick(item));
        col.appendChild(el);
      }
    }

    grid.appendChild(col);
  }

  container.appendChild(grid);
  return date;
}
