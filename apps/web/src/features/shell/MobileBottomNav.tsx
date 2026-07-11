import { Languages, MoreHorizontal, Settings } from 'lucide-react';
import { useState } from 'react';
import type { Locale } from '../../lib/api';
import { AppRoute, routeHref } from '../../lib/appRoutes';
import { initials } from '../../lib/crudFormUi';
import type { Translator } from '../../lib/translator';
import {
  isShellMoreRoute,
  SHELL_BOTTOM_NAV_ROUTES,
  SHELL_MORE_NAV,
  SHELL_PRIMARY_NAV,
  SHELL_REPORTING_NAV,
  shellNavItemIsActive,
} from './shellNav';

export function MobileBottomNav({
  locale,
  navigate,
  route,
  setLocale,
  t,
  userName,
}: {
  locale: Locale;
  navigate: (route: AppRoute) => void;
  route: AppRoute;
  setLocale: (locale: Locale) => void;
  t: Translator;
  userName: string;
}) {
  const [moreOpen, setMoreOpen] = useState(false);
  const bottomItems = [
    ...SHELL_PRIMARY_NAV,
    SHELL_REPORTING_NAV[0],
  ];

  function goTo(nextRoute: AppRoute) {
    navigate(nextRoute);
    setMoreOpen(false);
  }

  return (
    <>
      {moreOpen ? (
        <button
          aria-label={t('cancel')}
          className="mobile-nav-scrim"
          onClick={() => setMoreOpen(false)}
          type="button"
        />
      ) : null}
      <nav aria-label={t('nav')} className="mobile-bottom-nav">
        {bottomItems.map((item) => {
          const Icon = item.icon;
          const active = shellNavItemIsActive(route, item.route);
          return (
            <a
              aria-current={active ? 'page' : undefined}
              className={active ? 'active' : ''}
              href={routeHref(item.route)}
              key={item.route}
              onClick={(event) => {
                event.preventDefault();
                goTo(item.route);
              }}
            >
              <Icon aria-hidden="true" />
              <span>{t(item.labelKey)}</span>
            </a>
          );
        })}
        <button
          aria-expanded={moreOpen}
          className={moreOpen || isShellMoreRoute(route) ? 'active' : ''}
          onClick={() => setMoreOpen((open) => !open)}
          type="button"
        >
          <MoreHorizontal aria-hidden="true" />
          <span>{t('navMore')}</span>
        </button>
      </nav>
      {moreOpen ? (
        <div className="mobile-nav-more-panel" role="dialog" aria-label={t('navMore')}>
          <div className="mobile-nav-more-section">
            <span>{t('reporting')}</span>
            {SHELL_REPORTING_NAV.map((item) => (
              <a
                className={route === item.route ? 'active' : ''}
                href={routeHref(item.route)}
                key={item.route}
                onClick={(event) => {
                  event.preventDefault();
                  goTo(item.route);
                }}
              >
                {t(item.labelKey)}
              </a>
            ))}
          </div>
          <div className="mobile-nav-more-section">
            <span>{t('manage')}</span>
            {SHELL_MORE_NAV.filter((item) => item.group === 'manage').map((item) => (
              <a
                className={route === item.route ? 'active' : ''}
                href={routeHref(item.route)}
                key={item.route}
                onClick={(event) => {
                  event.preventDefault();
                  goTo(item.route);
                }}
              >
                {t(item.labelKey)}
              </a>
            ))}
          </div>
          <div className="mobile-nav-more-section">
            <span>{t('admin')}</span>
            {SHELL_MORE_NAV.filter((item) => item.group === 'admin').map((item) => (
              <a
                className={route === item.route ? 'active' : ''}
                href={routeHref(item.route)}
                key={item.route}
                onClick={(event) => {
                  event.preventDefault();
                  goTo(item.route);
                }}
              >
                {t(item.labelKey)}
              </a>
            ))}
          </div>
          <div className="mobile-nav-more-footer">
            <button type="button" title={t('language')} onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
              <Languages aria-hidden="true" />
              <span>{t('language')}</span>
            </button>
            <a
              className={route === 'profile' ? 'active' : ''}
              href={routeHref('profile')}
              onClick={(event) => {
                event.preventDefault();
                goTo('profile');
              }}
            >
              <Settings aria-hidden="true" />
              <span>{t('profileSettings')}</span>
            </a>
            <div className="profile-avatar" aria-hidden="true">
              {initials(userName)}
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
