import type { LucideIcon } from 'lucide-react';
import {
  BarChart3,
  Building2,
  CalendarDays,
  Clock3,
  FileText,
  FolderKanban,
  Import,
  LayoutDashboard,
  ListTodo,
  MessageSquareText,
  Settings,
  Tags,
} from 'lucide-react';
import type { AppRoute } from '../../lib/appRoutes';
import type { MessageKey } from '../../lib/i18n';

export type ShellNavGroup = 'primary' | 'reporting' | 'manage' | 'admin';

export type ShellNavItem = {
  route: AppRoute;
  labelKey: MessageKey;
  icon: LucideIcon;
  group: ShellNavGroup;
};

export const SHELL_PRIMARY_NAV: ShellNavItem[] = [
  { route: 'dashboard', labelKey: 'dashboard', icon: LayoutDashboard, group: 'primary' },
  { route: 'timesheet', labelKey: 'time', icon: Clock3, group: 'primary' },
  { route: 'calendar', labelKey: 'calendar', icon: CalendarDays, group: 'primary' },
];

export const SHELL_REPORTING_NAV: ShellNavItem[] = [
  { route: 'overview', labelKey: 'reporting', icon: BarChart3, group: 'reporting' },
  { route: 'detailed', labelKey: 'detailed', icon: BarChart3, group: 'reporting' },
  { route: 'daily-summary', labelKey: 'dailySummary', icon: MessageSquareText, group: 'reporting' },
];

export const SHELL_MANAGE_NAV: ShellNavItem[] = [
  { route: 'projects', labelKey: 'projects', icon: FolderKanban, group: 'manage' },
  { route: 'tasks', labelKey: 'tasks', icon: ListTodo, group: 'manage' },
  { route: 'clients', labelKey: 'clients', icon: Building2, group: 'manage' },
  { route: 'tags', labelKey: 'tags', icon: Tags, group: 'manage' },
];

export const SHELL_ADMIN_NAV: ShellNavItem[] = [
  { route: 'import-export', labelKey: 'importExport', icon: Import, group: 'admin' },
  { route: 'invoices', labelKey: 'invoices', icon: FileText, group: 'admin' },
  { route: 'settings', labelKey: 'settings', icon: Settings, group: 'admin' },
];

export const SHELL_BOTTOM_NAV_ROUTES: AppRoute[] = ['dashboard', 'timesheet', 'calendar', 'overview'];

export const SHELL_MORE_NAV: ShellNavItem[] = [
  ...SHELL_REPORTING_NAV.filter((item) => item.route === 'detailed' || item.route === 'daily-summary'),
  ...SHELL_MANAGE_NAV,
  ...SHELL_ADMIN_NAV,
];

export function isReportingRoute(route: AppRoute): boolean {
  return route === 'overview' || route === 'detailed' || route === 'daily-summary';
}

export function isShellMoreRoute(route: AppRoute): boolean {
  return SHELL_MORE_NAV.some((item) => item.route === route) || route === 'profile';
}

export function shellNavItemIsActive(route: AppRoute, itemRoute: AppRoute): boolean {
  if (itemRoute === 'overview') {
    return isReportingRoute(route);
  }
  return route === itemRoute;
}
