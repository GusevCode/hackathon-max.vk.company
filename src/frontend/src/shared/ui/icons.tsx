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
