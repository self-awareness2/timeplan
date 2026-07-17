import { api } from '../api';
import type { ScheduleItem, StatsData } from '../types';
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

  renderAnalytics(container, data);

  if (data.execution) {
    const executionTitle = document.createElement('h3');
    executionTitle.textContent = '执行结果';
    executionTitle.style.margin = '18px 0 12px';
    container.appendChild(executionTitle);
    const executionChart = document.createElement('div');
    executionChart.className = 'bar-chart card';
    executionChart.style.padding = '16px';
    const labels: Record<string, string> = { notStarted: '未开始', running: '执行中', executed: '按计划执行', delayed: '延迟执行', skipped: '未执行' };
    const entries = Object.entries(data.execution).filter(([, value]) => value > 0);
    const max = Math.max(1, ...entries.map(([, value]) => value));
    if (entries.length === 0) executionChart.appendChild(chartEmpty('暂无执行数据'));
    for (const [key, value] of entries) {
      const row = document.createElement('div');
      row.className = 'bar-row';
      row.innerHTML = `<span>${labels[key] ?? escapeHtml(key)}</span><div class="bar-track"><div class="bar-fill execution-fill" style="width:${(value / max) * 100}%"></div></div><span>${value}</span>`;
      executionChart.appendChild(row);
    }
    container.appendChild(executionChart);
  }

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

function renderAnalytics(container: HTMLElement, data: StatsData): void {
  const section = document.createElement('section');
  section.className = 'analytics-grid';
  const execution = data.execution ?? {};
  const pie = chartCard('执行结果占比');
  const pieValues = [
    { label: '按计划完成', value: execution.executed ?? 0, color: '#22a06b' },
    { label: '执行中', value: execution.running ?? 0, color: '#3268d6' },
    { label: '未完成', value: execution.skipped ?? 0, color: '#e45858' },
    { label: '未开始', value: execution.notStarted ?? 0, color: '#94a3b8' },
  ].filter((item) => item.value > 0);
  pie.appendChild(donut(pieValues));
  section.appendChild(pie);

  const waterfall = chartCard('计划与实际用时');
  waterfall.appendChild(waterfallChart(data.daily ?? []));
  section.appendChild(waterfall);

  const trend = chartCard('近期待办执行趋势');
  trend.appendChild(trendChart(data.daily ?? []));
  section.appendChild(trend);

  const reasons = chartCard('未完成原因');
  reasons.appendChild(reasonList(data.byReason ?? {}));
  section.appendChild(reasons);
  container.appendChild(section);
}

function chartCard(title: string): HTMLElement {
  const card = document.createElement('div');
  card.className = 'card analytics-card';
  const heading = document.createElement('h3');
  heading.textContent = title;
  card.appendChild(heading);
  return card;
}

function donut(values: Array<{ label: string; value: number; color: string }>): HTMLElement {
  const wrap = document.createElement('div');
  wrap.className = 'donut-layout';
  const total = values.reduce((sum, item) => sum + item.value, 0);
  const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('viewBox', '0 0 42 42'); svg.classList.add('donut-chart');
  const track = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
  track.setAttribute('cx', '21'); track.setAttribute('cy', '21'); track.setAttribute('r', '15.9');
  track.setAttribute('fill', 'none'); track.setAttribute('stroke', 'var(--panel-soft)'); track.setAttribute('stroke-width', '5');
  svg.appendChild(track);
  let offset = 0;
  for (const item of values) {
    const circle = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
    const portion = total ? (item.value / total) * 100 : 0;
    circle.setAttribute('cx', '21'); circle.setAttribute('cy', '21'); circle.setAttribute('r', '15.9');
    circle.setAttribute('fill', 'none'); circle.setAttribute('stroke', item.color); circle.setAttribute('stroke-width', '5');
    circle.setAttribute('stroke-dasharray', `${portion} ${100 - portion}`); circle.setAttribute('stroke-dashoffset', `${25 - offset}`);
    svg.appendChild(circle); offset += portion;
  }
  const center = document.createElementNS('http://www.w3.org/2000/svg', 'text');
  center.setAttribute('x', '21'); center.setAttribute('y', '22.5'); center.setAttribute('text-anchor', 'middle'); center.setAttribute('class', 'donut-total'); center.textContent = String(total); svg.appendChild(center);
  wrap.appendChild(svg);
  const legend = document.createElement('div'); legend.className = 'chart-legend';
  if (!values.length) legend.appendChild(chartEmpty('暂无执行数据'));
  for (const item of values) { const line = document.createElement('div'); line.innerHTML = `<i style="background:${item.color}"></i><span>${escapeHtml(item.label)}</span><b>${item.value}</b>`; legend.appendChild(line); }
  wrap.appendChild(legend); return wrap;
}

function waterfallChart(daily: NonNullable<StatsData['daily']>): HTMLElement {
  if (!daily.length || daily.every((item) => item.plannedMinutes === 0 && item.actualMinutes === 0)) return chartEmpty('暂无用时数据');
  const wrap = document.createElement('div'); wrap.className = 'waterfall-chart';
  const max = Math.max(1, ...daily.flatMap((item) => [item.plannedMinutes, item.actualMinutes]));
  for (const item of daily.slice(-7)) { const row = document.createElement('div'); row.className = 'waterfall-row'; row.innerHTML = `<span>${item.date.slice(5)}</span><div class="waterfall-bars"><i class="planned" style="height:${Math.max(4, item.plannedMinutes / max * 100)}%"></i><i class="actual" style="height:${Math.max(4, item.actualMinutes / max * 100)}%"></i></div><b>${item.actualMinutes}/${item.plannedMinutes}m</b>`; wrap.appendChild(row); }
  return wrap;
}

function trendChart(daily: NonNullable<StatsData['daily']>): HTMLElement {
  if (!daily.length || daily.every((item) => item.executed === 0 && item.skipped === 0)) return chartEmpty('暂无执行趋势数据');
  const wrap = document.createElement('div'); wrap.className = 'trend-chart';
  const max = Math.max(1, ...daily.map((item) => item.executed + item.skipped));
  for (const item of daily.slice(-7)) { const col = document.createElement('div'); col.className = 'trend-column'; col.innerHTML = `<div class="trend-stack"><i class="trend-done" style="height:${item.executed / max * 100}%"></i><i class="trend-skipped" style="height:${item.skipped / max * 100}%"></i></div><span>${item.date.slice(5)}</span>`; wrap.appendChild(col); }
  return wrap;
}

function reasonList(reasons: Record<string, number>): HTMLElement {
  const wrap = document.createElement('div'); wrap.className = 'reason-list';
  const entries = Object.entries(reasons).sort((a, b) => b[1] - a[1]);
  if (!entries.length) { wrap.classList.add('chart-empty'); wrap.textContent = '暂无未完成原因数据'; return wrap; }
  const max = entries[0][1];
  for (const [label, count] of entries.slice(0, 5)) { const row = document.createElement('div'); row.innerHTML = `<span>${escapeHtml(label)}</span><div><i style="width:${count / max * 100}%"></i></div><b>${count}</b>`; wrap.appendChild(row); }
  return wrap;
}

function chartEmpty(message: string): HTMLElement {
  const empty = document.createElement('div');
  empty.className = 'chart-empty';
  empty.textContent = message;
  return empty;
}
