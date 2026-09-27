import { apiStateLabel } from "../../../entities/service-status/model/types";
import { useApiHealth } from "../../../features/check-api/model/useApiHealth";
import { useAppVersion } from "../../../features/show-version/model/useAppVersion";
import { ArrowIcon } from "../../../shared/ui/icons";
import { SiteHeader } from "../../../widgets/site-header/ui/SiteHeader";
import { StatusBoard } from "../../../widgets/status-board/ui/StatusBoard";

export function StatusPage() {
  const { state, isRefreshing, refresh } = useApiHealth();
  const version = useAppVersion();
  const botUrl = import.meta.env.VITE_MAX_BOT_URL?.trim();
  const isOnline = state === "online";
  const protocolLabel = window.location.protocol === "https:" ? "Подключено" : "Локальный режим";
  const signalText = state === "checking"
    ? "Проверяем систему"
    : isOnline
      ? "Система в норме"
      : "Требуется внимание";

  return (
    <div className="app-shell">
      <SiteHeader />
      <main className="main-content">
        <section className="hero" aria-labelledby="page-title">
          <div className={`signal ${state === "offline" ? "is-offline" : ""}`} aria-hidden="true">
            <span className="signal-dot" />
            <span>{signalText}</span>
          </div>
          <h1 id="page-title">Сервис запущен</h1>
          <p>Go backend, Docker и автоматический деплой работают на одном защищённом контуре.</p>
          <div className="version-badge" aria-label={`Версия приложения: ${version}`}>
            <span>DEPLOYED VERSION</span>
            <code>{version}</code>
          </div>
        </section>

        <StatusBoard apiState={state} protocolLabel={protocolLabel} apiLabel={apiStateLabel(state)} />

        <div className="actions">
          <button type="button" className="primary-action" onClick={() => void refresh()} disabled={isRefreshing}>
            {isRefreshing ? "Проверяем…" : "Проверить API"}
            <ArrowIcon />
          </button>
          {botUrl ? (
            <a className="secondary-action" href={botUrl} target="_blank" rel="noreferrer">
              Открыть бота в MAX
            </a>
          ) : null}
        </div>
        <p className="sr-only" aria-live="polite">Статус Go API: {apiStateLabel(state)}</p>
      </main>
      <footer className="site-footer">
        <span>MAX HACKATHON / 2026</span>
        <span className="footer-domain">max.conspiracy-team.ru</span>
      </footer>
    </div>
  );
}
