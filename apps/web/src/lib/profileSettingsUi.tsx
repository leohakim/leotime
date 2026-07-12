import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, Save, Settings, UserRound } from 'lucide-react';
import { FormEvent, useEffect, useMemo, useRef, useState } from 'react';
import {
  changePassword,
  fetchProfile,
  isApiError,
  mapApiFieldErrors,
  updateProfile,
  type ChangePasswordInput,
  type LayoutMode,
  type Locale,
  type Profile,
  type ProfileUpdateInput,
  type SessionResponse,
  type ThemeMode,
  type User,
} from './api';
import {
  getExperiencePresetDimensions,
  type ExperiencePreset,
  type NamedExperiencePreset,
  type NavigationMode,
} from './experience';
import { ExperienceSwitcher } from './experienceUi';
import type { MessageKey } from './i18n';
import { useToast } from './toast';

export type Translator = (key: MessageKey) => string;

const TIMEZONES = [
  'Europe/Madrid',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'America/Argentina/Buenos_Aires',
  'America/Mexico_City',
  'America/Sao_Paulo',
  'Asia/Tokyo',
  'Australia/Sydney',
];

const CURRENCIES = ['EUR', 'USD', 'GBP', 'ARS', 'MXN', 'BRL', 'CHF', 'CAD'];

type ProfileFormState = ProfileUpdateInput;

type PasswordFormState = {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
};

type ProfileFormErrors = Partial<Record<keyof ProfileFormState | 'form', string>>;
type PasswordFormErrors = Partial<Record<keyof PasswordFormState | 'form', string>>;

function buildFormFromProfile(profile: Profile): ProfileFormState {
  return {
    name: profile.name,
    email: profile.email,
    locale: profile.locale,
    layoutMode: profile.layoutMode,
    taskProjectRequired: profile.settings.taskProjectRequired,
    defaultCurrency: profile.settings.defaultCurrency,
    timezone: profile.settings.timezone,
    themeMode: profile.settings.themeMode,
    timerStillRunningEnabled: profile.settings.timerStillRunningEnabled,
    timerStillRunningHours: profile.settings.timerStillRunningHours,
    backupEmailOnSuccess: profile.settings.backupEmailOnSuccess,
    backupEmailOnFailure: profile.settings.backupEmailOnFailure,
    restoreEmailOnSuccess: profile.settings.restoreEmailOnSuccess,
    restoreEmailOnFailure: profile.settings.restoreEmailOnFailure,
  };
}

function buildFormFromUser(user: User, themeMode: ThemeMode): ProfileFormState {
  return {
    name: user.name,
    email: user.email,
    locale: user.locale,
    layoutMode: user.layoutMode,
    taskProjectRequired: false,
    defaultCurrency: 'EUR',
    timezone: 'Europe/Madrid',
    themeMode,
    timerStillRunningEnabled: true,
    timerStillRunningHours: 8,
    backupEmailOnSuccess: false,
    backupEmailOnFailure: true,
    restoreEmailOnSuccess: false,
    restoreEmailOnFailure: true,
  };
}

function fieldClass(error?: string) {
  return error ? 'form-field has-error' : 'form-field';
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) {
    return null;
  }
  return (
    <span className="field-message" id={id} role="alert">
      {message}
    </span>
  );
}

export function ProfileSettingsPanel({
  focusSection,
  layoutMode,
  navigationMode,
  onApplyExperiencePreset,
  preset,
  setLayoutMode,
  setLocale,
  setNavigationMode,
  setThemeMode,
  t,
  themeMode,
  user,
}: {
  focusSection?: 'settings';
  layoutMode: LayoutMode;
  navigationMode: NavigationMode;
  onApplyExperiencePreset: (preset: NamedExperiencePreset) => void;
  preset: ExperiencePreset;
  setLayoutMode: (layoutMode: LayoutMode) => void;
  setLocale: (locale: Locale) => void;
  setNavigationMode: (navigationMode: NavigationMode) => void;
  setThemeMode: (themeMode: ThemeMode) => void;
  t: Translator;
  themeMode: ThemeMode;
  user: User;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const profileQuery = useQuery({
    queryKey: ['profile'],
    queryFn: fetchProfile,
    retry: false,
  });
  const serverHydratedRef = useRef(false);
  const [form, setForm] = useState<ProfileFormState>(() => buildFormFromUser(user, themeMode));
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const [passwordForm, setPasswordForm] = useState<PasswordFormState>({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  });
  const [passwordErrors, setPasswordErrors] = useState<PasswordFormErrors>({});

  const timezoneOptions = useMemo(() => {
    if (form.timezone && !TIMEZONES.includes(form.timezone)) {
      return [form.timezone, ...TIMEZONES];
    }
    return TIMEZONES;
  }, [form.timezone]);

  useEffect(() => {
    if (focusSection === 'settings') {
      document.getElementById('settings')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }, [focusSection]);

  useEffect(() => {
    if (!profileQuery.data || serverHydratedRef.current) {
      return;
    }
    serverHydratedRef.current = true;
    setForm(buildFormFromProfile(profileQuery.data));
    setLocale(profileQuery.data.locale);
    setLayoutMode(profileQuery.data.layoutMode);
    setThemeMode(profileQuery.data.settings.themeMode);
  }, [profileQuery.data, setLayoutMode, setLocale, setThemeMode]);

  const updateMutation = useMutation({
    mutationFn: updateProfile,
    onSuccess: (profile) => {
      queryClient.setQueryData(['profile'], profile);
      queryClient.setQueryData(['session'], (current: SessionResponse | undefined) => {
        if (!current?.user) {
          return current;
        }
        return {
          ...current,
          user: {
            ...current.user,
            name: profile.name,
            email: profile.email,
            locale: profile.locale,
            layoutMode: profile.layoutMode,
          },
        };
      });
      setForm(buildFormFromProfile(profile));
      setLocale(profile.locale);
      setLayoutMode(profile.layoutMode);
      setThemeMode(profile.settings.themeMode);
      setErrors({});
      toast.success(t('profileSaved'));
    },
    onError: (error) => {
      const fieldErrors = mapApiFieldErrors<keyof ProfileFormState>(error, {
        name: 'name',
        email: 'email',
        defaultCurrency: 'defaultCurrency',
        timezone: 'timezone',
        timerStillRunningHours: 'timerStillRunningHours',
      });
      setErrors({
        ...fieldErrors,
        form: Object.keys(fieldErrors).length > 0 ? undefined : isApiError(error) ? error.message : t('profileSaveFailed'),
      });
      toast.error(isApiError(error) ? error.message : t('profileSaveFailed'));
    },
  });

  const passwordMutation = useMutation({
    mutationFn: changePassword,
    onSuccess: () => {
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
      setPasswordErrors({});
      queryClient.clear();
      void queryClient.invalidateQueries({ queryKey: ['session'] });
      toast.success(t('passwordChanged'));
    },
    onError: (error) => {
      const fieldErrors = mapApiFieldErrors<keyof PasswordFormState>(error, {
        currentPassword: 'currentPassword',
        newPassword: 'newPassword',
      });
      setPasswordErrors({
        ...fieldErrors,
        form: Object.keys(fieldErrors).length > 0 ? undefined : isApiError(error) ? error.message : t('passwordChangeFailed'),
      });
      toast.error(isApiError(error) ? error.message : t('passwordChangeFailed'));
    },
  });

  function updateField<K extends keyof ProfileFormState>(key: K, value: ProfileFormState[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    setErrors((current) => ({ ...current, [key]: undefined, form: undefined }));
    if (key === 'layoutMode') {
      setLayoutMode(value as LayoutMode);
    }
    if (key === 'themeMode') {
      setThemeMode(value as ThemeMode);
    }
  }

  function applyExperiencePresetForForm(value: NamedExperiencePreset) {
    onApplyExperiencePreset(value);
    const dimensions = getExperiencePresetDimensions(value);
    setForm((current) => ({
      ...current,
      themeMode: dimensions.themeMode,
      layoutMode: dimensions.layoutMode,
    }));
  }

  function updatePasswordField<K extends keyof PasswordFormState>(key: K, value: PasswordFormState[K]) {
    setPasswordForm((current) => ({ ...current, [key]: value }));
    setPasswordErrors((current) => ({ ...current, [key]: undefined, form: undefined }));
  }

  function validateProfile(next: ProfileFormState): ProfileFormErrors {
    const nextErrors: ProfileFormErrors = {};
    if (next.name.trim().length < 2) {
      nextErrors.name = t('profileNameRequired');
    }
    if (!next.email.trim()) {
      nextErrors.email = t('profileEmailRequired');
    }
    if (!/^[A-Z]{3}$/.test(next.defaultCurrency.trim().toUpperCase())) {
      nextErrors.defaultCurrency = t('profileCurrencyInvalid');
    }
    if (next.timerStillRunningHours < 1 || next.timerStillRunningHours > 24) {
      nextErrors.timerStillRunningHours = t('profileTimerStillRunningHoursInvalid');
    }
    return nextErrors;
  }

  function validatePassword(next: PasswordFormState): PasswordFormErrors {
    const nextErrors: PasswordFormErrors = {};
    if (!next.currentPassword) {
      nextErrors.currentPassword = t('profileCurrentPasswordRequired');
    }
    if (next.newPassword.length < 8) {
      nextErrors.newPassword = t('profileNewPasswordRequired');
    }
    if (next.newPassword !== next.confirmPassword) {
      nextErrors.confirmPassword = t('profilePasswordMismatch');
    }
    return nextErrors;
  }

  function submitProfile(event: FormEvent) {
    event.preventDefault();
    const nextErrors = validateProfile(form);
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      return;
    }
    updateMutation.mutate({
      ...form,
      name: form.name.trim(),
      email: form.email.trim(),
      defaultCurrency: form.defaultCurrency.trim().toUpperCase(),
      timezone: form.timezone.trim() || 'Europe/Madrid',
      timerStillRunningHours: Math.trunc(form.timerStillRunningHours),
    });
  }

  function submitPassword(event: FormEvent) {
    event.preventDefault();
    const nextErrors = validatePassword(passwordForm);
    if (Object.keys(nextErrors).length > 0) {
      setPasswordErrors(nextErrors);
      return;
    }
    passwordMutation.mutate({
      currentPassword: passwordForm.currentPassword,
      newPassword: passwordForm.newPassword,
    });
  }

  return (
    <section className="clients-section profile-settings-section" id="profile" aria-labelledby="profile-title">
      <div className="clients-heading">
        <div className="section-title-group">
          <span className="section-kicker">
            <UserRound aria-hidden="true" />
            {t('profileSettings')}
          </span>
          <h2 id="profile-title">{t('profilePanelTitle')}</h2>
          <p>{t('profilePanelSubtitle')}</p>
        </div>
        {profileQuery.isLoading ? (
          <span className="sync-pill">{t('loading')}</span>
        ) : profileQuery.isError ? (
          <span className="sync-pill warning-pill">{t('profileLoadDegraded')}</span>
        ) : (
          <span className="sync-pill">{t('synced')}</span>
        )}
      </div>

      {profileQuery.isError ? (
        <div className="form-alert profile-settings-alert" role="alert">
          <CircleAlert aria-hidden="true" />
          {t('profileLoadFailedHint')}
        </div>
      ) : null}

      <div className="profile-settings-grid">
        <form className="client-editor profile-settings-form" noValidate onSubmit={submitProfile}>
          <div className="editor-header" id="profile-section-account">
            <div>
              <span>{t('profileAccountSection')}</span>
              <h3>{t('profileAccountHeading')}</h3>
            </div>
          </div>

          {errors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {errors.form}
            </div>
          ) : null}

          <div className="client-form-grid profile-account-grid">
            <label className={fieldClass(errors.name)} htmlFor="profile-name">
              <span>
                {t('name')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.name ? 'profile-name-error' : undefined}
                id="profile-name"
                onChange={(event) => updateField('name', event.target.value)}
                value={form.name}
              />
              <FieldError id="profile-name-error" message={errors.name} />
            </label>

            <label className={fieldClass(errors.email)} htmlFor="profile-email">
              <span>
                {t('email')} <em>{t('required')}</em>
              </span>
              <input
                aria-describedby={errors.email ? 'profile-email-error' : undefined}
                id="profile-email"
                onChange={(event) => updateField('email', event.target.value)}
                type="email"
                value={form.email}
              />
              <FieldError id="profile-email-error" message={errors.email} />
            </label>
          </div>

          <div className="profile-settings-divider" id="settings" />

          <div className="editor-header">
            <div>
              <span className="section-kicker">
                <Settings aria-hidden="true" />
                {t('settings')}
              </span>
              <h3>{t('profilePreferencesHeading')}</h3>
            </div>
          </div>

          <div className="client-form-grid profile-preferences-grid">
            <label className="form-field" htmlFor="profile-locale">
              <span>{t('language')}</span>
              <select id="profile-locale" onChange={(event) => updateField('locale', event.target.value as Locale)} value={form.locale}>
                <option value="es">{t('languageEs')}</option>
                <option value="en">{t('languageEn')}</option>
              </select>
            </label>

            <div className="form-field profile-experience-field">
              <ExperienceSwitcher
                layoutMode={layoutMode}
                navigationMode={navigationMode}
                onApplyPreset={applyExperiencePresetForForm}
                preset={preset}
                setLayoutMode={(value) => updateField('layoutMode', value)}
                setNavigationMode={setNavigationMode}
                setThemeMode={(value) => updateField('themeMode', value)}
                themeMode={themeMode}
                t={t}
                variant="settings"
              />
            </div>

            <label className={fieldClass(errors.defaultCurrency)} htmlFor="profile-currency">
              <span>{t('defaultCurrency')}</span>
              <select
                id="profile-currency"
                onChange={(event) => updateField('defaultCurrency', event.target.value)}
                value={form.defaultCurrency}
              >
                {CURRENCIES.map((currency) => (
                  <option key={currency} value={currency}>
                    {currency}
                  </option>
                ))}
              </select>
              <FieldError id="profile-currency-error" message={errors.defaultCurrency} />
            </label>

            <label className="form-field" htmlFor="profile-timezone">
              <span>{t('timezone')}</span>
              <select id="profile-timezone" onChange={(event) => updateField('timezone', event.target.value)} value={form.timezone}>
                {timezoneOptions.map((timezone) => (
                  <option key={timezone} value={timezone}>
                    {timezone}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <div className="settings-toggle-row profile-behavior-row">
            <input
              checked={form.taskProjectRequired}
              id="profile-task-project-required"
              onChange={(event) => updateField('taskProjectRequired', event.target.checked)}
              type="checkbox"
            />
            <label htmlFor="profile-task-project-required">{t('profileTaskProjectRequired')}</label>
          </div>

          <div className="profile-settings-divider" />

          <div className="editor-header profile-notifications-header" id="profile-section-notifications">
            <div>
              <span>{t('profileEmailNotificationsSection')}</span>
              <h3>{t('profileEmailNotificationsHeading')}</h3>
            </div>
          </div>

          <div className="profile-notifications-panel">
            <p className="profile-notification-subheading">{t('profileTimerNotificationsHeading')}</p>

            <div className="settings-toggle-row">
              <input
                checked={form.timerStillRunningEnabled}
                id="profile-timer-still-running-enabled"
                onChange={(event) => updateField('timerStillRunningEnabled', event.target.checked)}
                type="checkbox"
              />
              <label htmlFor="profile-timer-still-running-enabled">{t('profileTimerStillRunningEnabled')}</label>
            </div>

            <div className={`settings-inline-control ${errors.timerStillRunningHours ? 'has-error' : ''}`}>
              <label htmlFor="profile-timer-still-running-hours">{t('profileTimerStillRunningHours')}</label>
              <input
                className="settings-compact-input"
                disabled={!form.timerStillRunningEnabled}
                id="profile-timer-still-running-hours"
                min={1}
                max={24}
                onChange={(event) => updateField('timerStillRunningHours', Number(event.target.value))}
                step={1}
                type="number"
                value={form.timerStillRunningHours}
              />
              <FieldError id="profile-timer-still-running-hours-error" message={errors.timerStillRunningHours} />
            </div>

            <p className="profile-notification-subheading">{t('profileBackupNotificationsHeading')}</p>

            <div className="settings-toggle-row">
              <input
                checked={form.backupEmailOnSuccess}
                id="profile-backup-email-success"
                onChange={(event) => updateField('backupEmailOnSuccess', event.target.checked)}
                type="checkbox"
              />
              <label htmlFor="profile-backup-email-success">{t('profileBackupEmailOnSuccess')}</label>
            </div>

            <div className="settings-toggle-row">
              <input
                checked={form.backupEmailOnFailure}
                id="profile-backup-email-failure"
                onChange={(event) => updateField('backupEmailOnFailure', event.target.checked)}
                type="checkbox"
              />
              <label htmlFor="profile-backup-email-failure">{t('profileBackupEmailOnFailure')}</label>
            </div>

            <div className="settings-toggle-row">
              <input
                checked={form.restoreEmailOnSuccess}
                id="profile-restore-email-success"
                onChange={(event) => updateField('restoreEmailOnSuccess', event.target.checked)}
                type="checkbox"
              />
              <label htmlFor="profile-restore-email-success">{t('profileRestoreEmailOnSuccess')}</label>
            </div>

            <div className="settings-toggle-row">
              <input
                checked={form.restoreEmailOnFailure}
                id="profile-restore-email-failure"
                onChange={(event) => updateField('restoreEmailOnFailure', event.target.checked)}
                type="checkbox"
              />
              <label htmlFor="profile-restore-email-failure">{t('profileRestoreEmailOnFailure')}</label>
            </div>
          </div>

          <div className="client-form-actions">
            <button disabled={updateMutation.isPending} type="submit">
              <Save aria-hidden="true" />
              {updateMutation.isPending ? t('loading') : t('saveProfile')}
            </button>
          </div>
        </form>

        <form className="client-editor profile-password-form" id="profile-section-password" noValidate onSubmit={submitPassword}>
          <div className="editor-header">
            <div>
              <span>{t('profilePasswordSection')}</span>
              <h3>{t('profilePasswordHeading')}</h3>
            </div>
          </div>

          {passwordErrors.form ? (
            <div className="form-alert" role="alert">
              <CircleAlert aria-hidden="true" />
              {passwordErrors.form}
            </div>
          ) : null}

          <div className="client-form-grid profile-password-grid">
            <label className={fieldClass(passwordErrors.currentPassword)} htmlFor="profile-current-password">
              <span>{t('profileCurrentPassword')}</span>
              <input
                autoComplete="current-password"
                id="profile-current-password"
                onChange={(event) => updatePasswordField('currentPassword', event.target.value)}
                type="password"
                value={passwordForm.currentPassword}
              />
              <FieldError id="profile-current-password-error" message={passwordErrors.currentPassword} />
            </label>

            <label className={fieldClass(passwordErrors.newPassword)} htmlFor="profile-new-password">
              <span>{t('profileNewPassword')}</span>
              <input
                autoComplete="new-password"
                id="profile-new-password"
                onChange={(event) => updatePasswordField('newPassword', event.target.value)}
                type="password"
                value={passwordForm.newPassword}
              />
              <FieldError id="profile-new-password-error" message={passwordErrors.newPassword} />
            </label>

            <label className={fieldClass(passwordErrors.confirmPassword)} htmlFor="profile-confirm-password">
              <span>{t('profileConfirmPassword')}</span>
              <input
                autoComplete="new-password"
                id="profile-confirm-password"
                onChange={(event) => updatePasswordField('confirmPassword', event.target.value)}
                type="password"
                value={passwordForm.confirmPassword}
              />
              <FieldError id="profile-confirm-password-error" message={passwordErrors.confirmPassword} />
            </label>
          </div>

          <div className="client-form-actions">
            <button disabled={passwordMutation.isPending} type="submit">
              <Save aria-hidden="true" />
              {passwordMutation.isPending ? t('loading') : t('changePassword')}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}
