import { useMutation } from '@tanstack/react-query';
import { ArrowLeft, Languages, Mail, Play, Save } from 'lucide-react';
import { FormEvent, useEffect, useState } from 'react';
import { login, isApiError, requestPasswordReset, resetPassword, type Locale } from './api';
import type { MessageKey } from './i18n';
import { LeotimeLogo } from './leotimeLogo';
import { useToast } from './toast';

export type AuthTranslator = (key: MessageKey) => string;

type AuthView = 'login' | 'forgot' | 'reset';

function parseAuthView(): { view: AuthView; resetToken: string } {
  const hash = window.location.hash.replace(/^#/, '');
  const [path, query = ''] = hash.split('?');
  if (path === 'reset-password') {
    return { view: 'reset', resetToken: new URLSearchParams(query).get('token') ?? '' };
  }
  return { view: 'login', resetToken: '' };
}

export function AuthScreen({
  locale,
  onAuthenticated,
  setLocale,
  t,
}: {
  locale: Locale;
  onAuthenticated: () => void;
  setLocale: (locale: Locale) => void;
  t: AuthTranslator;
}) {
  const toast = useToast();
  const [view, setView] = useState<AuthView>(() => parseAuthView().view);
  const [resetToken, setResetToken] = useState(() => parseAuthView().resetToken);
  const [email, setEmail] = useState('admin@example.com');
  const [password, setPassword] = useState('change-me-now');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');

  useEffect(() => {
    function syncAuthView() {
      const parsed = parseAuthView();
      setView(parsed.view);
      setResetToken(parsed.resetToken);
    }

    window.addEventListener('hashchange', syncAuthView);
    syncAuthView();
    return () => window.removeEventListener('hashchange', syncAuthView);
  }, []);

  const loginMutation = useMutation({
    mutationFn: () => login(email, password),
    onSuccess: () => {
      onAuthenticated();
    },
    onError: (error) => {
      toast.error(isApiError(error) && error.code === 'invalid_credentials' ? t('loginFailed') : isApiError(error) ? error.message : t('loginFailed'));
    },
  });

  const forgotMutation = useMutation({
    mutationFn: () => requestPasswordReset(email),
    onSuccess: () => {
      toast.success(t('resetLinkSent'));
      setView('login');
      window.location.hash = '';
    },
    onError: () => toast.error(t('passwordResetRequestFailed')),
  });

  const resetMutation = useMutation({
    mutationFn: () => resetPassword(resetToken, newPassword),
    onSuccess: () => {
      toast.success(t('passwordResetSuccess'));
      setPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setView('login');
      window.location.hash = '';
    },
    onError: () => toast.error(t('passwordResetFailed')),
  });

  function submitLogin(event: FormEvent) {
    event.preventDefault();
    loginMutation.mutate();
  }

  function submitForgot(event: FormEvent) {
    event.preventDefault();
    forgotMutation.mutate();
  }

  function submitReset(event: FormEvent) {
    event.preventDefault();
    if (!resetToken) {
      toast.error(t('passwordResetFailed'));
      return;
    }
    if (newPassword.length < 8) {
      toast.error(t('profileNewPasswordRequired'));
      return;
    }
    if (newPassword !== confirmPassword) {
      toast.error(t('profilePasswordMismatch'));
      return;
    }
    resetMutation.mutate();
  }

  return (
    <main className="login-screen">
      <section className="login-panel" aria-labelledby="auth-title">
        <LeotimeLogo className="brand-row" markSize={28} />
        <h1 id="auth-title">
          {view === 'forgot' ? t('forgotPasswordTitle') : view === 'reset' ? t('resetPasswordTitle') : t('welcome')}
        </h1>
        {view === 'forgot' ? <p className="login-help">{t('forgotPasswordHelp')}</p> : null}
        {view === 'reset' ? <p className="login-help">{t('resetPasswordHelp')}</p> : null}

        {view === 'login' ? (
          <form onSubmit={submitLogin} className="login-form">
            <label>
              {t('email')}
              <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" autoComplete="username" />
            </label>
            <label>
              {t('password')}
              <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" autoComplete="current-password" />
            </label>
            <button type="submit" disabled={loginMutation.isPending}>
              <Play aria-hidden="true" />
              {t('login')}
            </button>
            <button className="ghost-button login-inline-link" type="button" onClick={() => setView('forgot')}>
              {t('forgotPassword')}
            </button>
          </form>
        ) : null}

        {view === 'forgot' ? (
          <form onSubmit={submitForgot} className="login-form">
            <label>
              {t('email')}
              <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" autoComplete="username" />
            </label>
            <button type="submit" disabled={forgotMutation.isPending}>
              <Mail aria-hidden="true" />
              {t('sendResetLink')}
            </button>
            <button className="ghost-button login-inline-link" type="button" onClick={() => { setView('login'); window.location.hash = ''; }}>
              <ArrowLeft aria-hidden="true" />
              {t('backToLogin')}
            </button>
          </form>
        ) : null}

        {view === 'reset' ? (
          <form onSubmit={submitReset} className="login-form">
            <label>
              {t('profileNewPassword')}
              <input value={newPassword} onChange={(event) => setNewPassword(event.target.value)} type="password" autoComplete="new-password" />
            </label>
            <label>
              {t('profileConfirmPassword')}
              <input value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} type="password" autoComplete="new-password" />
            </label>
            <button type="submit" disabled={resetMutation.isPending || !resetToken}>
              <Save aria-hidden="true" />
              {t('resetPasswordSubmit')}
            </button>
            <button className="ghost-button login-inline-link" type="button" onClick={() => { setView('login'); window.location.hash = ''; }}>
              <ArrowLeft aria-hidden="true" />
              {t('backToLogin')}
            </button>
          </form>
        ) : null}

        <button className="ghost-button" type="button" onClick={() => setLocale(locale === 'es' ? 'en' : 'es')}>
          <Languages aria-hidden="true" />
          {t('language')}
        </button>
      </section>
    </main>
  );
}
