import type { MessageKey } from './i18n';

export type Translator = (key: MessageKey) => string;

const SETTINGS_SECTIONS = [
  { id: 'profile-section-account', labelKey: 'profileAccountSection' },
  { id: 'settings', labelKey: 'settings' },
  { id: 'profile-section-notifications', labelKey: 'profileEmailNotificationsSection' },
  { id: 'profile-section-password', labelKey: 'profilePasswordSection' },
  { id: 'backups', labelKey: 'backupHeading' },
] as const satisfies ReadonlyArray<{ id: string; labelKey: MessageKey }>;

export function scrollToSettingsSection(sectionId: string) {
  document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

export function SettingsSectionNav({ t }: { t: Translator }) {
  return (
    <nav aria-label={t('settingsSectionNavLabel')} className="settings-section-nav">
      {SETTINGS_SECTIONS.map((section) => (
        <button
          className="settings-section-nav-item"
          key={section.id}
          onClick={() => scrollToSettingsSection(section.id)}
          type="button"
        >
          {t(section.labelKey)}
        </button>
      ))}
    </nav>
  );
}
