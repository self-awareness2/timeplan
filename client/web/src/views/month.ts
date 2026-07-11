import { api, parseDateParts } from '../api';
import type { ScheduleItem } from '../types';
import { renderEmpty, renderItemCard } from './common';

export async function renderMonthView(
  container: HTMLElement,
  year: number,
  month: number,
  onItemClick: (item: ScheduleItem) => void,
  onDaySelect?: (date: string) => void
): Promise<{ year: number; month: number }> {
  const data = await api.listMonth(year, month);
  container.innerHTML = '';

  const cal = document.createElement('div');
  cal.className = 'month-calendar';

  for (const label of ['日', '一', '二', '三', '四', '五', '六']) {
    const head = document.createElement('div');
    head.className = 'cal-head';
    head.textContent = label;
    cal.appendChild(head);
  }

  for (let i = 0; i < data.firstWeekday; ++i) {
    const empty = document.createElement('div');
    empty.className = 'cal-cell empty';
    cal.appendChild(empty);
  }

  for (const day of data.days) {
    const cell = document.createElement('div');
    cell.className = 'cal-cell' + (day.count > 0 ? ' has-items' : '');
    cell.innerHTML = `
      <div class="day-num">${parseDateParts(day.date).day}</div>
      ${day.count > 0 ? '<div class="dot"></div>' : ''}
    `;
    cell.addEventListener('click', () => {
      if (onDaySelect) onDaySelect(day.date);
    });
    cal.appendChild(cell);
  }

  container.appendChild(cal);

  const listTitle = document.createElement('h3');
  listTitle.textContent = `${data.monthName}全部计划 (${data.items.length})`;
  listTitle.style.marginBottom = '12px';
  container.appendChild(listTitle);

  const list = document.createElement('div');
  if (data.items.length === 0) {
    list.appendChild(renderEmpty());
  } else {
    for (const item of data.items) {
      list.appendChild(renderItemCard(item, onItemClick));
    }
  }
  container.appendChild(list);

  return { year: data.year, month: data.month };
}
