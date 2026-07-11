import { ChevronDown } from 'lucide-react';
import { AppRoute, routeHref } from '../../lib/appRoutes';
import type { Translator } from '../../lib/translator';
import {
  isReportingRoute,
  SHELL_ADMIN_NAV,
  SHELL_MANAGE_NAV,
  SHELL_PRIMARY_NAV,
  SHELL_REPORTING_NAV,
  shellNavItemIsActive,
} from './shellNav';

export function SidebarNav({
  navigate,
  route,
  t,
}: {
  navigate: (route: AppRoute) => void;
  route: AppRoute;
  t: Translator;
}) {
  return (
    <nav className="sidebar-nav" aria-label={t('time')}>
      {SHELL_PRIMARY_NAV.map((item) => {
        const Icon = item.icon;
        return (
          <a
            className={shellNavItemIsActive(route, item.route) ? 'active' : ''}
            href={routeHref(item.route)}
            key={item.route}
            onClick={(event) => {
              event.preventDefault();
              navigate(item.route);
            }}
          >
            <Icon aria-hidden="true" />
            <span>{t(item.labelKey)}</span>
          </a>
        );
      })}

      <a
        className={`nav-parent${isReportingRoute(route) ? ' active' : ''}`}
        href={routeHref('overview')}
        onClick={(event) => {
          event.preventDefault();
          navigate('overview');
        }}
      >
        {(() => {
          const Icon = SHELL_REPORTING_NAV[0].icon;
          return <Icon aria-hidden="true" />;
        })()}
        <span>{t('reporting')}</span>
        <ChevronDown aria-hidden="true" />
      </a>
      <div className="nav-children" aria-label={t('reporting')}>
        {SHELL_REPORTING_NAV.map((item) => (
          <a
            className={route === item.route ? 'active' : ''}
            href={routeHref(item.route)}
            key={item.route}
            onClick={(event) => {
              event.preventDefault();
              navigate(item.route);
            }}
          >
            {t(item.labelKey)}
          </a>
        ))}
      </div>

      <span className="nav-section-label">{t('manage')}</span>
      {SHELL_MANAGE_NAV.map((item) => {
        const Icon = item.icon;
        return (
          <a
            className={route === item.route ? 'active' : ''}
            href={routeHref(item.route)}
            key={item.route}
            onClick={(event) => {
              event.preventDefault();
              navigate(item.route);
            }}
          >
            <Icon aria-hidden="true" />
            <span>{t(item.labelKey)}</span>
          </a>
        );
      })}

      <span className="nav-section-label">{t('admin')}</span>
      {SHELL_ADMIN_NAV.map((item) => {
        const Icon = item.icon;
        return (
          <a
            className={route === item.route ? 'active' : ''}
            href={routeHref(item.route)}
            key={item.route}
            onClick={(event) => {
              event.preventDefault();
              navigate(item.route);
            }}
          >
            <Icon aria-hidden="true" />
            <span>{t(item.labelKey)}</span>
          </a>
        );
      })}
    </nav>
  );
}
