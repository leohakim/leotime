import { Play, Square } from 'lucide-react';

export function TimerStopIcon({ className }: { className?: string }) {
  return <Square aria-hidden className={className} fill="currentColor" strokeWidth={0} />;
}

export function TimerPlayIcon({ className }: { className?: string }) {
  return <Play aria-hidden className={className} fill="currentColor" strokeWidth={0} />;
}
