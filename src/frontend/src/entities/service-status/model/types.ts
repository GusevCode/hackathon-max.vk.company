export type ApiState = "checking" | "online" | "offline";

export function apiStateLabel(state: ApiState): string {
  if (state === "checking") return "Проверяем";
  if (state === "online") return "Доступен";
  return "Недоступен";
}
