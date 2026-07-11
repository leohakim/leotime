import { ChevronDown, Languages, Settings } from 'lucide-react';
import type { Locale } from '../../lib/api';
import { AppRoute, routeHref } from '../../lib/appRoutes';
import { initials } from '../../lib/crudFormUi';
import { LeotimeMark } from '../../lib/leotimeLogo';
import type { TimeEntry } from '../../lib/api';
import { SidebarTimer } from '../../lib/timerUi';
import type { Translator } from '../../lib/translator';
import { SidebarNav } from './SidebarNav';

export function ShellSidebar({
  activeTimer,
  locale,
  navigate,
  onStop,
  route,
  setLocale,
  stoppingTimerId,
  t,
  userName,
}: {
  activeTimer: TimeEntry | null;
  locale: Locale;
  navigate: (route: AppRoute) => void;
  onStop: (timeEntryId: string) => void;
  route: AppRoute;
  setLocale: (locale: Locale) => void;
  stoppingTimerId: string | null;
  t: Translator;
  userName: string;
}) {
  return (
    <aside className="sidebar shell-sidebar" aria-label="Primary">
      <div className="org-switcher">
        <LeotimeMark className="org-avatar-logo" size={30} title="leotime" />
        <span>{t('organizationName')}</span>
        <ChevronDown aria-hidden="true" />
      </div>

      <SidebarTimer
        activeTimer={activeTimer}
        onStop={onStop}
        stoppingTimerId={stoppingTimerId}
        t={t}
      />

      <SidebarNav navigate={navigate} route={route} t={t} />

      <div className="sidebar-footer shell-sidebar-footer">
        <button type="button" title={t('language')} onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
          <Languages aria-hidden="true" />
        </button>
        <a
          className={route === 'profile' ? 'active' : ''}
          href={routeHref('profile')}
          onClick={(event) => {
            event.preventDefault();
            navigate('profile');
          }}
        >
          <Settings aria-hidden="true" />
          <span>{t('profileSettings')}</span>
        </a>
        <div className="profile-avatar" aria-hidden="true">
          {initials(userName)}
        </div>
      </div>
    </aside>
  );
}
