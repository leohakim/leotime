import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;

export const SETTINGS_SECTIONS = [
  { id: 'profile-section-account', labelKey: 'profileAccountSection' },
  { id: 'settings', labelKey: 'settings' },
  { id: 'profile-section-notifications', labelKey: 'profileEmailNotificationsSection' },
  { id: 'profile-section-password', labelKey: 'profilePasswordSection' },
  { id: 'ai-summary-settings', labelKey: 'aiSettingsHeading' },
  { id: 'vcs-settings', labelKey: 'vcsHeading' },
  { id: 'backups', labelKey: 'backupHeading' },
] as const satisfies ReadonlyArray<{ id: string; labelKey: MessageKey }>;

export type SettingsSectionId = (typeof SETTINGS_SECTIONS)[number]['id'];

export function isSettingsSectionId(value: string): value is SettingsSectionId {
  return SETTINGS_SECTIONS.some((section) => section.id === value);
}

export function defaultSettingsSection(route: 'settings' | 'profile'): SettingsSectionId {
  return route === 'settings' ? 'settings' : 'profile-section-account';
}

/** @deprecated Prefer selecting a section via SettingsSectionNav onSelect. */
export function scrollToSettingsSection(sectionId: string) {
  document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

export function SettingsSectionNav({
  activeSection,
  onSelect,
  t,
}: {
  activeSection: SettingsSectionId;
  onSelect: (section: SettingsSectionId) => void;
  t: Translator;
}) {
  return (
    <nav aria-label={t('settingsSectionNavLabel')} className="settings-section-nav">
      <p className="settings-section-nav-title">{t('settingsSectionNavLabel')}</p>
      <div className="settings-section-nav-list" role="tablist" aria-orientation="vertical">
        {SETTINGS_SECTIONS.map((section) => {
          const selected = activeSection === section.id;
          return (
            <button
              aria-selected={selected}
              className={selected ? 'settings-section-nav-item is-active' : 'settings-section-nav-item'}
              key={section.id}
              onClick={() => onSelect(section.id)}
              role="tab"
              type="button"
            >
              {t(section.labelKey)}
            </button>
          );
        })}
      </div>
    </nav>
  );
}
