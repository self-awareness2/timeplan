import { api } from '../api';
import type { ScheduleItem } from '../types';
import { renderEmpty, renderItemCard } from './common';

const HOUR_HEIGHT = 48;

function timeToMinutes(time: string): number {
  const [hour, minute] = time.split(':').map(Number);
  if (!Number.isFinite(hour) || !Number.isFinite(minute)) return 0;
  return Math.max(0, Math.min(24 * 60, hour * 60 + minute));
}

function todayString(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

function currentTimeLabel(date: Date): string {
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

export async function renderDayView(
  container: HTMLElement,
  date: string,
  onItemClick: (item: ScheduleItem) => void,
  onTimeRangeSelect?: (startTime: string, endTime: string) => void
): Promise<string> {
  const data = await api.listDay(date);
  container.innerHTML = '';

  const timed: ScheduleItem[] = [];
  const allDay: ScheduleItem[] = [];
  for (const item of data.items) {
    if (item.hasTime) timed.push(item);
    else allDay.push(item);
  }

  const layout = document.createElement('div');
  layout.className = 'day-layout';

  const timeline = document.createElement('div');
  timeline.className = 'timeline';
  for (let h = 0; h < 24; h++) {
    const hour = document.createElement('div');
    hour.className = 'timeline-hour';
    hour.textContent = `${String(h).padStart(2, '0')}:00`;
    timeline.appendChild(hour);
  }

  const content = document.createElement('div');

  if (allDay.length > 0) {
    const section = document.createElement('div');
    section.className = 'all-day-section';
    section.innerHTML = '<h3>全天</h3>';
    for (const item of allDay) section.appendChild(renderItemCard(item, onItemClick));
    content.appendChild(section);
  }

  const list = document.createElement('div');
  list.className = 'day-items';
  if (timed.length === 0 && allDay.length === 0) {
    list.appendChild(renderEmpty());
  } else {
    for (const item of timed) {
      const card = renderItemCard(item, onItemClick);
      card.classList.add('timeline-item');

      const start = timeToMinutes(item.startTime);
      const end = item.endTime && item.endTime !== '00:00' ? timeToMinutes(item.endTime) : start + 60;
      const duration = Math.max(30, end - start);

      card.style.top = `${(start / 60) * HOUR_HEIGHT}px`;
      card.style.minHeight = `${Math.max(44, (duration / 60) * HOUR_HEIGHT - 8)}px`;
      list.appendChild(card);
    }
  }
  content.appendChild(list);

  layout.appendChild(timeline);
  layout.appendChild(content);
  container.appendChild(layout);

  if (date === todayString()) {
    const now = new Date();
    const minutes = now.getHours() * 60 + now.getMinutes();
    const marker = document.createElement('div');
    marker.className = 'current-time-marker';
    marker.style.top = `${(minutes / 60) * HOUR_HEIGHT}px`;
    marker.innerHTML = `<span>${currentTimeLabel(now)}</span>`;
    list.appendChild(marker);
  }

  if (onTimeRangeSelect) enableRangeSelection(list, onTimeRangeSelect);

  return data.date;
}

function enableRangeSelection(
  list: HTMLElement,
  onTimeRangeSelect: (startTime: string, endTime: string) => void
): void {
  let startY = 0;
  let selecting = false;
  const selection = document.createElement('div');
  selection.className = 'time-selection';

  const yToMinutes = (clientY: number) => {
    const rect = list.getBoundingClientRect();
    const raw = ((clientY - rect.top + list.scrollTop) / HOUR_HEIGHT) * 60;
    const snapped = Math.round(raw / 15) * 15;
    return Math.max(0, Math.min(24 * 60, snapped));
  };

  const render = (a: number, b: number) => {
    const top = Math.min(a, b);
    const bottom = Math.max(a, b);
    selection.style.top = `${(top / 60) * HOUR_HEIGHT}px`;
    selection.style.height = `${Math.max(12, ((bottom - top) / 60) * HOUR_HEIGHT)}px`;
    selection.textContent = `${minutesToTime(top)} - ${minutesToTime(Math.max(bottom, top + 15))}`;
  };

  list.addEventListener('pointerdown', (e) => {
    if (e.pointerType === 'touch' || window.matchMedia('(pointer: coarse)').matches) return;
    if ((e.target as HTMLElement).closest('.item-card, .empty-state, button, input, select, textarea, a')) return;
    e.preventDefault();
    selecting = true;
    startY = yToMinutes(e.clientY);
    render(startY, startY + 30);
    list.appendChild(selection);
    list.setPointerCapture(e.pointerId);
  });

  list.addEventListener('pointermove', (e) => {
    if (!selecting) return;
    render(startY, yToMinutes(e.clientY));
  });

  list.addEventListener('pointerup', (e) => {
    if (!selecting) return;
    selecting = false;
    const endY = yToMinutes(e.clientY);
    selection.remove();
    const start = Math.min(startY, endY);
    const end = Math.max(startY, endY);
    if (Math.abs(end - start) < 15) return;
    onTimeRangeSelect(minutesToTime(start), minutesToTime(Math.min(24 * 60 - 1, end)));
  });

  list.addEventListener('pointercancel', () => {
    selecting = false;
    selection.remove();
  });
}

function minutesToTime(total: number): string {
  const clamped = Math.max(0, Math.min(23 * 60 + 59, total));
  const hour = Math.floor(clamped / 60);
  const minute = clamped % 60;
  return `${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}`;
}
