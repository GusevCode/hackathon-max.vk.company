import type { ReactNode } from "react";

type ServiceStatusItemProps = {
  label: string;
  value: string;
  icon: ReactNode;
  online?: boolean;
};

export function ServiceStatusItem({ label, value, icon, online = false }: ServiceStatusItemProps) {
  return (
    <div className="status-item">
      <span className={`status-icon ${online ? "is-online" : ""}`}>{icon}</span>
      <span className="status-copy">
        <span className="status-key">{label}</span>
        <strong>{value}</strong>
      </span>
      <span className={`service-state ${online ? "is-online" : ""}`}>
        <span className="service-state-dot" />
        {online ? "Работает" : "Проверка"}
      </span>
    </div>
  );
}
