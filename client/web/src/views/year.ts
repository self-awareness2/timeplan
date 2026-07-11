import { api } from '../api';
import type { ScheduleItem } from '../types';
import { escapeHtml, renderEmpty, renderItemCard } from './common';

export async function renderYearView(
  container: HTMLElement,
  year: number,
  onItemClick: (item: ScheduleItem) => void,
  onMonthSelect?: (year: number, month: number) => void
): Promise<number> {
  const data = await api.listYear(year);
  container.innerHTML = '';

  const summary = document.createElement('p');
  summary.style.marginBottom = '16px';
  summary.textContent = `${year} 年共 ${data.totalCount} 项计划`;
  container.appendChild(summary);

  const grid = document.createElement('div');
  grid.className = 'year-grid';

  for (const m of data.months) {
    const card = document.createElement('div');
    card.className = 'card year-month-card';
    card.innerHTML = `
      <h3><span>${m.monthName}</span><span class="tag">${m.count} 项</span></h3>
    `;

    if (m.items.length === 0) {
      card.appendChild(renderEmpty('无计划'));
    } else {
      const ul = document.createElement('div');
      for (const item of m.items.slice(0, 5)) {
        const row = document.createElement('div');
        row.style.cssText = 'font-size:0.875rem;padding:4px 0;cursor:pointer';
        row.innerHTML = `<span style="color:var(--text-muted)">${item.date.slice(5)}</span> ${escapeHtml(item.title)}`;
        row.addEventListener('click', (e) => {
          e.stopPropagation();
          onItemClick(item);
        });
        ul.appendChild(row);
      }
      if (m.items.length > 5) {
        const more = document.createElement('div');
        more.style.cssText = 'font-size:0.8rem;color:var(--text-muted)';
        more.textContent = `还有 ${m.items.length - 5} 项...`;
        ul.appendChild(more);
      }
      card.appendChild(ul);
    }

    card.addEventListener('click', () => {
      if (onMonthSelect) onMonthSelect(year, m.month);
    });
    grid.appendChild(card);
  }

  container.appendChild(grid);
  return data.year;
}

export async function renderSearchView(
  container: HTMLElement,
  keyword: string,
  onItemClick: (item: ScheduleItem) => void
): Promise<void> {
  container.innerHTML = '';
  if (!keyword.trim()) {
    container.appendChild(renderEmpty('请输入关键词搜索'));
    return;
  }

  const data = await api.search(keyword);
  const title = document.createElement('p');
  title.textContent = `搜索「${keyword}」找到 ${data.items.length} 项`;
  title.style.marginBottom = '12px';
  container.appendChild(title);

  if (data.items.length === 0) {
    container.appendChild(renderEmpty('未找到匹配计划'));
    return;
  }

  for (const item of data.items) {
    container.appendChild(renderItemCard(item, onItemClick));
  }
}

export async function renderStatsView(container: HTMLElement): Promise<void> {
  container.innerHTML = '';
  const data = await api.stats();

  const grid = document.createElement('div');
  grid.className = 'stats-grid';
  const stats = [
    { num: data.pending, label: '待办' },
    { num: data.inProgress, label: '进行中' },
    { num: data.completed, label: '已完成' },
    { num: data.cancelled, label: '已取消' },
    { num: data.total, label: '总计' },
  ];
  for (const s of stats) {
    const card = document.createElement('div');
    card.className = 'card stat-card';
    card.innerHTML = `<div class="num">${s.num}</div><div class="label">${s.label}</div>`;
    grid.appendChild(card);
  }
  container.appendChild(grid);

  const catTitle = document.createElement('h3');
  catTitle.textContent = '分类分布';
  catTitle.style.marginBottom = '12px';
  container.appendChild(catTitle);

  const entries = Object.entries(data.byCategory);
  if (entries.length === 0) {
    container.appendChild(renderEmpty('暂无分类数据'));
    return;
  }

  const max = Math.max(...entries.map(([, v]) => v));
  const chart = document.createElement('div');
  chart.className = 'bar-chart card';
  chart.style.padding = '16px';

  for (const [cat, count] of entries) {
    const row = document.createElement('div');
    row.className = 'bar-row';
    row.innerHTML = `
      <span>${escapeHtml(cat)}</span>
      <div class="bar-track"><div class="bar-fill" style="width:${(count / max) * 100}%"></div></div>
      <span>${count}</span>
    `;
    chart.appendChild(row);
  }
  container.appendChild(chart);
}
