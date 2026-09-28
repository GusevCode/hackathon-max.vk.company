import type { ReactNode } from "react";

function Icon({ children }: { children: ReactNode }) {
  return <svg viewBox="0 0 24 24" aria-hidden="true">{children}</svg>;
}

export function LockIcon() {
  return <Icon><path d="M7.5 10V7.75a4.5 4.5 0 0 1 9 0V10M6 10h12v9H6zM12 13.5v2" /></Icon>;
}

export function ServerIcon() {
  return <Icon><rect x="4" y="4" width="16" height="6" rx="1.5" /><rect x="4" y="14" width="16" height="6" rx="1.5" /><path d="M7 7h.01M7 17h.01M10 7h7M10 17h7" /></Icon>;
}

export function MessageIcon() {
  return <Icon><path d="M5 5h14v10H9l-4 4z" /><path d="M8 9h8M8 12h5" /></Icon>;
}

export function ArrowIcon() {
  return <Icon><path d="M5 12h13M13 6l6 6-6 6" /></Icon>;
}

export function RefreshIcon() {
  return <Icon><path d="M20 11a8 8 0 0 0-14.9-3.9L4 9" /><path d="M4 4v5h5" /><path d="M4 13a8 8 0 0 0 14.9 3.9L20 15" /><path d="M20 20v-5h-5" /></Icon>;
}

export function DatabaseIcon() {
  return <Icon><ellipse cx="12" cy="6" rx="7" ry="3" /><path d="M5 6v6c0 1.7 3.1 3 7 3s7-1.3 7-3V6" /><path d="M5 12v6c0 1.7 3.1 3 7 3s7-1.3 7-3v-6" /></Icon>;
}

export function GridIcon() {
  return <Icon><rect x="4" y="4" width="6" height="6" rx="1" /><rect x="14" y="4" width="6" height="6" rx="1" /><rect x="4" y="14" width="6" height="6" rx="1" /><rect x="14" y="14" width="6" height="6" rx="1" /></Icon>;
}

export function UsersIcon() {
  return <Icon><circle cx="9" cy="8" r="3" /><path d="M3.5 19a5.5 5.5 0 0 1 11 0" /><path d="M16 5.5a3 3 0 0 1 0 5.8M17 14a4.8 4.8 0 0 1 3.5 4.6" /></Icon>;
}

export function ChartIcon() {
  return <Icon><path d="M4 19V5M4 19h16" /><path d="m7 15 3-4 3 2 5-7" /></Icon>;
}
