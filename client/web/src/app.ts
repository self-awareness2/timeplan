import { api, addDays, parseDateParts, todayString } from './api';
import { showItemModal } from './modal';
import type { ScheduleItem, ViewName } from './types';
import { renderDayView } from './views/day';
import { renderWeekView } from './views/week';
import { renderMonthView } from './views/month';
import { renderYearView, renderSearchView, renderStatsView } from './views/year';

interface AppState {
  view: ViewName;
  currentDate: string;
  year: number;
  month: number;
  searchKeyword: string;
}

type IconName =
  | 'calendar'
  | 'chevron-left'
  | 'chevron-right'
  | 'clock'
  | 'log-out'
  | 'moon'
  | 'plus'
  | 'search'
  | 'sun'
  | 'bar-chart'
  | 'calendar-days'
  | 'calendar-range'
  | 'calendar-fold'
  | 'calendar-clock';

export class App {
  private state: AppState = {
    view: 'day',
    currentDate: todayString(),
    year: new Date().getFullYear(),
    month: new Date().getMonth() + 1,
    searchKeyword: '',
  };

  private root: HTMLElement;
  private mainEl!: HTMLElement;
  private metaEl!: HTMLElement;
  private statsEl!: HTMLElement;

  constructor(root: HTMLElement) {
    this.root = root;
    this.renderShell();
  }

  async init(): Promise<void> {
    await this.refreshHeader();
    await this.renderView();
  }

  private renderShell(): void {
    this.root.innerHTML = `
      <div class="app-shell">
        <header class="header">
          <h1>Chrona 时序</h1>
          <div class="header-meta" id="header-meta"></div>
          <div class="header-stats" id="header-stats"></div>
          <div class="header-search">
            <input type="search" id="search-input" placeholder="搜索计划..." />
            <button class="btn" id="search-btn">${iconSvg('search')}<span>搜索</span></button>
          </div>
          <button class="btn auth-logout" id="logout-btn">${iconSvg('log-out')}<span>退出</span></button>
          <button class="theme-toggle" id="theme-toggle" title="切换主题" aria-label="切换主题">${iconSvg('sun')}</button>
        </header>
        <nav class="sidebar" id="sidebar"></nav>
        <main class="main" id="main"></main>
        <button class="mobile-fab" id="mobile-add-btn" aria-label="添加计划">${iconSvg('plus')}</button>
        <nav class="bottom-nav" id="bottom-nav"></nav>
      </div>
    `;

    this.mainEl = this.root.querySelector('#main')!;
    this.metaEl = this.root.querySelector('#header-meta')!;
    this.statsEl = this.root.querySelector('#header-stats')!;
    this.buildNav(this.root.querySelector('#sidebar')!);
    this.buildNav(this.root.querySelector('#bottom-nav')!);

    this.root.querySelector('#search-btn')?.addEventListener('click', () => this.doSearch());
    this.root.querySelector('#search-input')?.addEventListener('keydown', (e) => {
      if ((e as KeyboardEvent).key === 'Enter') this.doSearch();
    });
    this.root.querySelector('#mobile-add-btn')?.addEventListener('click', () => this.openAddModal());
    this.root.querySelector('#theme-toggle')?.addEventListener('click', () => this.toggleTheme());

    const logoutBtn = this.root.querySelector('#logout-btn') as HTMLButtonElement | null;
    if (logoutBtn) {
      logoutBtn.style.display = api.isDesktop() ? 'none' : '';
      logoutBtn.addEventListener('click', () => {
        api.logout();
        window.location.reload();
      });
    }
  }

  private buildNav(container: HTMLElement): void {
    const items: { view: ViewName; label: string; icon: IconName }[] = [
      { view: 'day', label: '每日', icon: 'calendar-clock' },
      { view: 'week', label: '每周', icon: 'calendar-range' },
      { view: 'month', label: '每月', icon: 'calendar-days' },
      { view: 'year', label: '每年', icon: 'calendar-fold' },
      { view: 'search', label: '搜索', icon: 'search' },
      { view: 'stats', label: '统计', icon: 'bar-chart' },
    ];

    for (const item of items) {
      const btn = document.createElement('button');
      btn.className = 'nav-btn' + (this.state.view === item.view ? ' active' : '');
      btn.dataset.view = item.view;
      btn.innerHTML = `<span class="nav-icon">${iconSvg(item.icon)}</span><span>${item.label}</span>`;
      btn.addEventListener('click', () => this.setView(item.view));
      container.appendChild(btn);
    }

    if (container.classList.contains('sidebar')) {
      const addBtn = document.createElement('button');
      addBtn.className = 'nav-btn primary';
      addBtn.innerHTML = `${iconSvg('plus')}<span>添加计划</span>`;
      addBtn.addEventListener('click', () => this.openAddModal());
      container.appendChild(addBtn);
    }
  }

  private toggleTheme(): void {
    const html = document.documentElement;
    const next = html.getAttribute('data-theme') === 'light' ? 'dark' : 'light';
    html.setAttribute('data-theme', next);
    localStorage.setItem('chrona_theme', next);
    const toggle = this.root.querySelector('#theme-toggle');
    if (toggle) toggle.innerHTML = iconSvg(next === 'light' ? 'sun' : 'moon');
  }

  private updateNavActive(): void {
    this.root.querySelectorAll('.nav-btn[data-view]').forEach((el) => {
      el.classList.toggle('active', (el as HTMLElement).dataset.view === this.state.view);
    });
  }

  private async refreshHeader(): Promise<void> {
    try {
      const info = await api.getToday();
      this.metaEl.textContent = `${info.date} ${info.weekday}`;
      this.statsEl.innerHTML = `
        <span class="stat-badge">待办 ${info.pending}</span>
        <span class="stat-badge">进行中 ${info.inProgress}</span>
        <span class="stat-badge">完成 ${info.completed}</span>
      `;
    } catch {
      this.metaEl.textContent = todayString();
    }
  }

  private viewTitle(): string {
    const titles: Record<ViewName, string> = {
      day: `每日计划 · ${formatDateWithWeekday(this.state.currentDate)}`,
      week: `每周计划 · ${formatDateWithWeekday(this.state.currentDate)}`,
      month: `每月计划 · ${this.state.year}-${String(this.state.month).padStart(2, '0')}`,
      year: `每年计划 · ${this.state.year}`,
      search: '搜索',
      stats: '统计概览',
    };
    return titles[this.state.view];
  }

  private async renderView(): Promise<void> {
    this.updateNavActive();
    const header = document.createElement('div');
    header.className = 'view-header';
    header.innerHTML = `<h2>${this.viewTitle()}</h2>`;

    const nav = document.createElement('div');
    nav.className = 'view-nav';
    if (this.state.view === 'day') {
      nav.appendChild(this.iconBtn('上一天', 'chevron-left', () => this.shiftDay(-1)));
      nav.appendChild(this.navBtn('今天', () => { this.state.currentDate = todayString(); this.renderView(); }, false, 'calendar'));
      nav.appendChild(this.iconBtn('下一天', 'chevron-right', () => this.shiftDay(1)));
      nav.appendChild(this.navBtn('添加', () => this.openAddModal(), true, 'plus'));
    } else if (this.state.view === 'week') {
      nav.appendChild(this.navBtn('上周', () => this.shiftDay(-7), false, 'chevron-left'));
      nav.appendChild(this.navBtn('本周', () => { this.state.currentDate = todayString(); this.renderView(); }, false, 'calendar'));
      nav.appendChild(this.navBtn('下周', () => this.shiftDay(7), false, 'chevron-right'));
    } else if (this.state.view === 'month') {
      nav.appendChild(this.iconBtn('上个月', 'chevron-left', () => this.shiftMonth(-1)));
      nav.appendChild(this.navBtn('本月', () => {
        const t = parseDateParts(todayString());
        this.state.year = t.year;
        this.state.month = t.month;
        this.renderView();
      }, false, 'calendar'));
      nav.appendChild(this.iconBtn('下个月', 'chevron-right', () => this.shiftMonth(1)));
    } else if (this.state.view === 'year') {
      nav.appendChild(this.iconBtn('上一年', 'chevron-left', () => { this.state.year -= 1; this.renderView(); }));
      nav.appendChild(this.navBtn('今年', () => { this.state.year = new Date().getFullYear(); this.renderView(); }, false, 'calendar'));
      nav.appendChild(this.iconBtn('下一年', 'chevron-right', () => { this.state.year += 1; this.renderView(); }));
    }

    header.appendChild(nav);
    this.mainEl.innerHTML = '';
    this.mainEl.appendChild(header);
    const content = document.createElement('div');
    this.mainEl.appendChild(content);

    const onItemClick = (item: ScheduleItem) => {
      showItemModal(item, this.state.currentDate, {
        onSave: () => { this.refreshHeader(); this.renderView(); },
        onDelete: () => { this.refreshHeader(); this.renderView(); },
      });
    };

    try {
      switch (this.state.view) {
        case 'day':
          await renderDayView(content, this.state.currentDate, onItemClick, (startTime, endTime) => this.openAddModal(startTime, endTime));
          if (this.state.currentDate === todayString()) requestAnimationFrame(() => this.scrollToCurrentTime());
          break;
        case 'week':
          await renderWeekView(content, this.state.currentDate, onItemClick);
          break;
        case 'month':
          await renderMonthView(content, this.state.year, this.state.month, onItemClick, (date) => {
            this.state.currentDate = date;
            this.setView('day');
          });
          break;
        case 'year':
          await renderYearView(content, this.state.year, onItemClick, (y, m) => {
            this.state.year = y;
            this.state.month = m;
            this.setView('month');
          });
          break;
        case 'search':
          await renderSearchView(content, this.state.searchKeyword, onItemClick);
          break;
        case 'stats':
          await renderStatsView(content);
          break;
      }
    } catch (err) {
      content.innerHTML = `<div class="empty-state">加载失败: ${err}</div>`;
    }
  }

  private navBtn(label: string, onClick: () => void, accent = false, icon?: IconName): HTMLButtonElement {
    const btn = document.createElement('button');
    btn.className = 'btn' + (accent ? ' btn-accent' : '');
    btn.innerHTML = `${icon ? iconSvg(icon) : ''}<span>${label}</span>`;
    btn.addEventListener('click', onClick);
    return btn;
  }

  private iconBtn(label: string, icon: IconName, onClick: () => void): HTMLButtonElement {
    const btn = document.createElement('button');
    btn.className = 'btn icon-btn';
    btn.title = label;
    btn.setAttribute('aria-label', label);
    btn.innerHTML = iconSvg(icon);
    btn.addEventListener('click', onClick);
    return btn;
  }

  private setView(view: ViewName): void {
    this.state.view = view;
    this.renderView();
  }

  private shiftDay(delta: number): void {
    this.state.currentDate = addDays(this.state.currentDate, delta);
    this.renderView();
  }

  private shiftMonth(delta: number): void {
    this.state.month += delta;
    if (this.state.month < 1) {
      this.state.month = 12;
      this.state.year -= 1;
    } else if (this.state.month > 12) {
      this.state.month = 1;
      this.state.year += 1;
    }
    this.renderView();
  }

  private doSearch(): void {
    const input = this.root.querySelector('#search-input') as HTMLInputElement;
    this.state.searchKeyword = input.value.trim();
    this.setView('search');
  }

  private openAddModal(startTime?: string, endTime?: string): void {
    showItemModal(null, this.state.currentDate, {
      onSave: () => { this.refreshHeader(); this.renderView(); },
      defaultStartTime: startTime,
      defaultEndTime: endTime,
    });
  }

  private scrollToCurrentTime(): void {
    const marker = this.mainEl.querySelector('.current-time-marker') as HTMLElement | null;
    if (!marker) return;
    const mainRect = this.mainEl.getBoundingClientRect();
    const markerRect = marker.getBoundingClientRect();
    const offset = Math.max(80, mainRect.height * 0.28);
    const target = this.mainEl.scrollTop + markerRect.top - mainRect.top - offset;
    this.mainEl.scrollTo({ top: Math.max(0, target), behavior: 'smooth' });
  }
}

function formatDateWithWeekday(date: string): string {
  const d = new Date(date + 'T12:00:00');
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  return `${date} ${weekdays[d.getDay()]}`;
}

function iconSvg(name: IconName): string {
  const paths: Record<IconName, string> = {
    'bar-chart': '<path d="M4 19V9"/><path d="M12 19V5"/><path d="M20 19v-8"/>',
    calendar: '<path d="M8 2v4M16 2v4M3 10h18"/><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01"/>',
    'calendar-clock': '<path d="M8 2v4M16 2v4M3 10h18"/><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M12 14v3l2 1"/>',
    'calendar-days': '<path d="M8 2v4M16 2v4M3 10h18"/><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01M16 18h.01"/>',
    'calendar-fold': '<path d="M8 2v4M16 2v4M3 10h18"/><path d="M21 15v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h10"/><path d="M21 15h-5a1 1 0 0 1-1-1V9"/>',
    'calendar-range': '<path d="M8 2v4M16 2v4M3 10h18"/><rect x="3" y="4" width="18" height="18" rx="2"/><path d="M7 14h5M14 18h3"/>',
    'chevron-left': '<path d="M15 18l-6-6 6-6"/>',
    'chevron-right': '<path d="M9 6l6 6-6 6"/>',
    clock: '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
    'log-out': '<path d="M10 17l5-5-5-5"/><path d="M15 12H3"/><path d="M21 19V5a2 2 0 0 0-2-2h-5"/>',
    moon: '<path d="M21 12.8A8.5 8.5 0 1 1 11.2 3 6.5 6.5 0 0 0 21 12.8z"/>',
    plus: '<path d="M12 5v14M5 12h14"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/>',
    sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41"/>',
  };
  return `<svg class="btn-icon" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${paths[name]}</svg>`;
}
