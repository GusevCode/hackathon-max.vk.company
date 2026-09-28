import { ServiceStatusItem } from "../../../entities/service-status/ui/ServiceStatusItem";
import type { ApiState } from "../../../entities/service-status/model/types";
import { DatabaseIcon, LockIcon, MessageIcon, ServerIcon } from "../../../shared/ui/icons";

type StatusBoardProps = {
  apiState: ApiState;
  protocolLabel: string;
  apiLabel: string;
};

export function StatusBoard({ apiState, protocolLabel, apiLabel }: StatusBoardProps) {
  return (
    <section className="status-card" aria-label="Статус сервисов">
      <ServiceStatusItem label="HTTPS" value={protocolLabel} icon={<LockIcon />} online={protocolLabel === "Подключено"} />
      <ServiceStatusItem label="Go API" value={apiLabel} icon={<ServerIcon />} online={apiState === "online"} />
      <ServiceStatusItem label="MAX бот" value="Webhook" icon={<MessageIcon />} online />
      <ServiceStatusItem label="Контур данных" value="Tarantool · SeaweedFS" icon={<DatabaseIcon />} online={apiState === "online"} />
    </section>
  );
}
