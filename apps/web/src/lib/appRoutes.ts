import { useCallback, useEffect, useState } from 'react';

export type AppRoute =
  | 'dashboard'
  | 'timesheet'
  | 'calendar'
  | 'overview'
  | 'detailed'
  | 'shared'
  | 'projects'
  | 'tasks'
  | 'clients'
  | 'tags'
  | 'import-export'
  | 'invoices'
  | 'settings'
  | 'profile'
  | 'manual-time-entry';

export const DEFAULT_ROUTE: AppRoute = 'timesheet';

const ROUTE_ALIASES: Record<string, AppRoute> = {
  dashboard: 'dashboard',
  timesheet: 'timesheet',
  calendar: 'calendar',
  reports: 'overview',
  overview: 'overview',
  detailed: 'detailed',
  shared: 'shared',
  projects: 'projects',
  tasks: 'tasks',
  clients: 'clients',
  tags: 'tags',
  'import-export': 'import-export',
  invoices: 'invoices',
  settings: 'settings',
  profile: 'profile',
  'manual-time-entry': 'manual-time-entry',
};

export function parseRoute(hash: string): AppRoute {
  const raw = hash.replace(/^#/, '').replace(/^\//, '').split('?')[0].trim();
  if (!raw) {
    return DEFAULT_ROUTE;
  }
  return ROUTE_ALIASES[raw] ?? DEFAULT_ROUTE;
}

export function routeHref(route: AppRoute): string {
  return `#${route}`;
}

export function navigateTo(route: AppRoute) {
  if (typeof window === 'undefined') {
    return;
  }
  const nextHash = routeHref(route);
  if (window.location.hash !== nextHash) {
    window.location.hash = route;
  }
}

export function useAppRoute(): [AppRoute, (route: AppRoute) => void] {
  const [route, setRoute] = useState<AppRoute>(() => parseRoute(window.location.hash));

  useEffect(() => {
    function syncRoute() {
      setRoute(parseRoute(window.location.hash));
    }

    window.addEventListener('hashchange', syncRoute);
    syncRoute();
    return () => window.removeEventListener('hashchange', syncRoute);
  }, []);

  const navigate = useCallback((next: AppRoute) => {
    navigateTo(next);
    setRoute(next);
  }, []);

  return [route, navigate];
}

export function routeUsesTimeEntries(route: AppRoute): boolean {
  return route === 'timesheet' || route === 'calendar' || route === 'manual-time-entry';
}

export function routeUsesTimesheetEntries(route: AppRoute): boolean {
  return route === 'timesheet' || route === 'calendar';
}

export function routeShowsTimerBar(route: AppRoute): boolean {
  return route === 'timesheet' || route === 'calendar' || route === 'manual-time-entry';
}
