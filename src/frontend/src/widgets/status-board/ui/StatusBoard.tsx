import { ServiceStatusItem } from "../../../entities/service-status/ui/ServiceStatusItem";
import type { ApiState } from "../../../entities/service-status/model/types";
import { LockIcon, MessageIcon, ServerIcon } from "../../../shared/ui/icons";

type StatusBoardProps = {
  apiState: ApiState;
  protocolLabel: string;
  apiLabel: string;
};

export function StatusBoard({ apiState, protocolLabel, apiLabel }: StatusBoardProps) {
  return (
    <section className="status-card" aria-label="Статус сервисов">
      <ServiceStatusItem label="HTTPS" value={protocolLabel} icon={<LockIcon />} />
      <ServiceStatusItem label="Go API" value={apiLabel} icon={<ServerIcon />} online={apiState === "online"} />
      <ServiceStatusItem label="MAX echo bot" value="Long Polling" icon={<MessageIcon />} />
    </section>
  );
}
