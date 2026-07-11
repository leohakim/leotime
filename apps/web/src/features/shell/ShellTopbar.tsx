import { LogOut, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';
import type { LayoutMode, ThemeMode } from '../../lib/api';
import { AppRoute } from '../../lib/appRoutes';
import type { ExperiencePreset, NamedExperiencePreset, NavigationMode } from '../../lib/experience';
import { ExperienceSwitcher } from '../../lib/experienceUi';
import { LeotimeMark } from '../../lib/leotimeLogo';
import { OfflineStatusPill } from '../../lib/offline/offlineStatusUi';
import type { Translator } from '../../lib/translator';

export function ShellTopbar({
  layoutMode,
  navigationMode,
  onApplyExperiencePreset,
  onLogout,
  pageTitle,
  preset,
  setLayoutMode,
  setNavigationMode,
  setThemeMode,
  themeMode,
  t,
}: {
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
  onApplyExperiencePreset: (preset: NamedExperiencePreset) => void;
  onLogout: () => void;
  pageTitle: string;
  preset: ExperiencePreset;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  setNavigationMode: (navigationMode: NavigationMode) => void;
  setThemeMode: (themeMode: ThemeMode) => void;
  themeMode: ThemeMode;
  t: Translator;
}) {
  const [experienceOpen, setExperienceOpen] = useState(false);

  return (
    <header className="tracker-topbar shell-topbar">
      <div className="tracker-title">
        <LeotimeMark size={18} />
        <h1>{pageTitle}</h1>
      </div>
      <div className="toolbar shell-toolbar">
        <OfflineStatusPill t={t} />
        <div className="shell-toolbar-desktop">
          <ExperienceSwitcher
            layoutMode={layoutMode}
            navigationMode={navigationMode}
            onApplyPreset={onApplyExperiencePreset}
            preset={preset}
            setLayoutMode={setLayoutMode}
            setNavigationMode={setNavigationMode}
            setThemeMode={setThemeMode}
            themeMode={themeMode}
            t={t}
          />
        </div>
        <button
          aria-expanded={experienceOpen}
          className="shell-toolbar-menu-button"
          onClick={() => setExperienceOpen((open) => !open)}
          title={t('experience')}
          type="button"
        >
          <MoreHorizontal aria-hidden="true" />
          <span>{t('experience')}</span>
        </button>
        <button type="button" title={t('logout')} onClick={onLogout}>
          <LogOut aria-hidden="true" />
        </button>
      </div>
      {experienceOpen ? (
        <div className="shell-toolbar-drawer">
          <ExperienceSwitcher
            layoutMode={layoutMode}
            navigationMode={navigationMode}
            onApplyPreset={onApplyExperiencePreset}
            preset={preset}
            setLayoutMode={setLayoutMode}
            setNavigationMode={setNavigationMode}
            setThemeMode={setThemeMode}
            themeMode={themeMode}
            t={t}
            variant="settings"
          />
        </div>
      ) : null}
    </header>
  );
}
