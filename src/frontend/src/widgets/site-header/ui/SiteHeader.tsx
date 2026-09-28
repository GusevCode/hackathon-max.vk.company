type SiteHeaderProps = {
  signalText: string;
  isOnline: boolean;
};

export function SiteHeader({ signalText, isOnline }: SiteHeaderProps) {
  return (
    <header className="site-header">
      <a className="brand" href="/" aria-label="ЖКХ Контроль — на главную">
        <span className="brand-mark">ЖК</span>
        <span className="brand-name">ЖКХ КОНТРОЛЬ</span>
      </a>
      <div className="topbar-meta">
        <span className={`topbar-status ${isOnline ? "is-online" : ""}`}>
          <span className="status-pulse" />
          {signalText}
        </span>
        <span className="topbar-date">MAX WEBHOOK · PRODUCTION</span>
      </div>
    </header>
  );
}
