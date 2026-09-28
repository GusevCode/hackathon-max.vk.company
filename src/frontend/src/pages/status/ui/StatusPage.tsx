import { apiStateLabel } from "../../../entities/service-status/model/types";
import { useApiHealth } from "../../../features/check-api/model/useApiHealth";
import { useAppVersion } from "../../../features/show-version/model/useAppVersion";
import { ChartIcon, GridIcon, RefreshIcon, UsersIcon } from "../../../shared/ui/icons";
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

  const shortVersion = version.length > 16 ? `${version.slice(0, 8)}…${version.slice(-6)}` : version;
  const activityItems = [
    { time: "Сейчас", title: isOnline ? "API отвечает" : "API требует внимания", copy: "Проверка через /api/healthz", tone: isOnline ? "success" : "warning" },
    { time: "При запуске", title: "MAX Webhook активен", copy: "События доставляются в control bot", tone: "success" },
    { time: "Контур данных", title: "Хранилища подключены", copy: "Tarantool · SeaweedFS", tone: "success" },
  ];

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Навигация">
        <div className="sidebar-brand"><span className="brand-mark">ЖК</span><span>ЖКХ КОНТРОЛЬ</span></div>
        <div className="sidebar-section-label">РАЗДЕЛЫ</div>
        <nav className="sidebar-nav">
          <button className="nav-item is-active" type="button"><GridIcon /><span>Обзор</span></button>
          <button className="nav-item" type="button" disabled title="Раздел появится на следующем этапе"><UsersIcon /><span>Пользователи</span></button>
          <button className="nav-item" type="button" disabled title="Раздел появится на следующем этапе"><ChartIcon /><span>Отчёты</span></button>
        </nav>
        <div className="sidebar-bottom">
          <span className="sidebar-version">CONTROL BOT / 2026</span>
          <span className="sidebar-domain">max.conspiracy-team.ru</span>
        </div>
      </aside>
      <div className="workspace">
        <SiteHeader signalText={signalText} isOnline={isOnline} />
        <main className="main-content">
          <div className="page-heading">
            <div>
              <span className="eyebrow">ОБЗОР СИСТЕМЫ</span>
              <h1 id="page-title">ЖКХ Контроль</h1>
              <p>Единый контур для контроля работ, фотоотчётов и команд MAX.</p>
            </div>
            <button type="button" className="refresh-action" onClick={() => void refresh()} disabled={isRefreshing}>
              <RefreshIcon />
              {isRefreshing ? "Проверяем…" : "Обновить статус"}
            </button>
          </div>

          <section className="overview-grid" aria-label="Сводка деплоя">
            <article className="deployment-card">
              <div className="card-kicker"><span className="live-dot" />DEPLOYED VERSION</div>
              <div className="deployment-version">{shortVersion}</div>
              <div className="deployment-meta"><span>Релиз приложения</span><code>{version}</code></div>
              <div className="deployment-footer"><span>Состояние релиза</span><strong>{isOnline ? "Система работает" : "Проверяем систему"}</strong></div>
            </article>
            <aside className="activity-card" aria-label="Последние события">
              <div className="card-heading"><h2>Последние события</h2><span className="activity-count">{activityItems.length}</span></div>
              <div className="activity-list">
                {activityItems.map((item) => (
                  <div className="activity-item" key={item.title}>
                    <span className={`activity-marker is-${item.tone}`} />
                    <div><div className="activity-time">{item.time}</div><strong>{item.title}</strong><p>{item.copy}</p></div>
                  </div>
                ))}
              </div>
            </aside>
          </section>

          <section className="services-section" aria-labelledby="services-title">
            <div className="section-heading"><div><span className="eyebrow">ИНФРАСТРУКТУРА</span><h2 id="services-title">Сервисы</h2></div><span className={`overall-state ${isOnline ? "is-online" : ""}`}><span className="service-state-dot" />{isOnline ? "Все системы в норме" : "Есть проблема"}</span></div>
            <StatusBoard apiState={state} protocolLabel={protocolLabel} apiLabel={apiStateLabel(state)} />
          </section>

          <div className="actions">
            {botUrl ? <a className="primary-action" href={botUrl} target="_blank" rel="noreferrer">Открыть бота в MAX <span>↗</span></a> : <span className="action-note">Ссылка на бота появится после настройки MAX_BOT_URL</span>}
          </div>
          <p className="sr-only" aria-live="polite">Статус Go API: {apiStateLabel(state)}</p>
        </main>
        <footer className="site-footer"><span>ЖКХ КОНТРОЛЬ / MAX WEBHOOK</span><span className="footer-domain">max.conspiracy-team.ru</span></footer>
      </div>
    </div>
  );
}
