import { useId } from 'react';

type LeotimeMarkProps = {
  className?: string;
  size?: number;
  title?: string;
};

export function LeotimeMark({ className, size = 24, title = 'leotime' }: LeotimeMarkProps) {
  const uid = useId().replace(/:/g, '');
  const bgId = `leotime-bg-${uid}`;
  const ringId = `leotime-ring-${uid}`;

  return (
    <svg
      aria-hidden={title ? undefined : true}
      className={className}
      fill="none"
      height={size}
      role={title ? 'img' : undefined}
      viewBox="0 0 32 32"
      width={size}
      xmlns="http://www.w3.org/2000/svg"
    >
      {title ? <title>{title}</title> : null}
      <rect fill={`url(#${bgId})`} height="32" rx="8" width="32" />
      <circle cx="16" cy="16" opacity="0.45" r="10" stroke={`url(#${ringId})`} strokeWidth="1.25" />
      <path d="M16 16V9.5" stroke="#5fb3d9" strokeLinecap="round" strokeWidth="2.25" />
      <path d="M16 16L21.25 12.75" stroke="#35c78a" strokeLinecap="round" strokeWidth="1.85" />
      <circle cx="16" cy="16" fill="#5fb3d9" r="1.65" />
      <path
        d="M16 6.5V7.75M16 24.25V25.5M25.5 16H24.25M7.75 16H6.5"
        stroke="#5fb3d9"
        strokeLinecap="round"
        strokeOpacity="0.55"
        strokeWidth="1.1"
      />
      <defs>
        <linearGradient id={bgId} x1="4" x2="28" y1="4" y2="28">
          <stop stopColor="#121318" />
          <stop offset="1" stopColor="#0c0d10" />
        </linearGradient>
        <linearGradient id={ringId} x1="6" x2="26" y1="6" y2="26">
          <stop stopColor="#5fb3d9" />
          <stop offset="1" stopColor="#35c78a" />
        </linearGradient>
      </defs>
    </svg>
  );
}

type LeotimeLogoProps = {
  className?: string;
  markSize?: number;
  showWordmark?: boolean;
};

export function LeotimeLogo({ className, markSize = 24, showWordmark = true }: LeotimeLogoProps) {
  return (
    <div className={className ? `leotime-logo ${className}` : 'leotime-logo'}>
      <LeotimeMark size={markSize} title="leotime" />
      {showWordmark ? <span className="leotime-wordmark">leotime</span> : null}
    </div>
  );
}
