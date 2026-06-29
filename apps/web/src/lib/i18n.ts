import type { Locale } from './api';

const messages = {
  es: {
    appName: 'leotime',
    calendar: 'Calendario',
    compact: 'Compacto',
    dashboard: 'Panel',
    email: 'Email',
    export: 'Exportar',
    invoices: 'Facturas',
    language: 'Idioma',
    layout: 'Distribucion',
    login: 'Entrar',
    logout: 'Salir',
    minimal: 'Minimalista',
    offlineReady: 'Offline listo',
    password: 'Contrasena',
    projects: 'Proyectos',
    reports: 'Informes',
    solid: 'Solid',
    start: 'Iniciar',
    stop: 'Parar',
    tags: 'Tags',
    tasks: 'Tareas',
    thisWeek: 'Esta semana',
    timer: 'Timer',
    timesheet: 'Timesheet',
    today: 'Hoy',
    trackWork: 'Registrar trabajo',
    welcome: 'Tu mesa de trabajo diaria',
  },
  en: {
    appName: 'leotime',
    calendar: 'Calendar',
    compact: 'Compact',
    dashboard: 'Dashboard',
    email: 'Email',
    export: 'Export',
    invoices: 'Invoices',
    language: 'Language',
    layout: 'Layout',
    login: 'Sign in',
    logout: 'Log out',
    minimal: 'Minimal',
    offlineReady: 'Offline ready',
    password: 'Password',
    projects: 'Projects',
    reports: 'Reports',
    solid: 'Solid',
    start: 'Start',
    stop: 'Stop',
    tags: 'Tags',
    tasks: 'Tasks',
    thisWeek: 'This week',
    timer: 'Timer',
    timesheet: 'Timesheet',
    today: 'Today',
    trackWork: 'Track work',
    welcome: 'Your daily workbench',
  },
} as const;

export type MessageKey = keyof typeof messages.es;

export function translate(locale: Locale, key: MessageKey): string {
  return messages[locale][key];
}

