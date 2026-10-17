import type { ReactNode } from 'react';
import { NavLink } from 'react-router';
import { cx } from '@/shared/lib/cx';
import { LcdDisplay, Led, ToggleSwitch } from '@/shared/ui';
import type { RackLayoutViewModel } from '../controller/useRackLayoutController';
import styles from './RackLayoutView.module.css';

export interface RackLayoutViewProps extends RackLayoutViewModel {
  children: ReactNode;
}

export function RackLayoutView({
  t,
  navItems,
  statusText,
  statusLed,
  lampOn,
  onLampChange,
  langs,
  lang,
  onLangChange,
  children,
}: RackLayoutViewProps) {
  return (
    <div className={styles.rack}>
      <div className={styles.side} aria-hidden />
      <div className={styles.column}>
        <header className={styles.receiver}>
          <div className={styles.brand}>
            <span className={styles.brandName}>Music Tracker</span>
            <span className={styles.brandModel}>{t.model}</span>
          </div>

          <div className={styles.status}>
            <Led state={statusLed} />
            <LcdDisplay label={t.serverStatus} size="sm">
              {statusText}
            </LcdDisplay>
          </div>

          <div className={styles.controls}>
            <fieldset className={styles.lang}>
              <legend className={styles.langLegend}>{t.language}</legend>
              <div className={styles.langKeys}>
                {langs.map((l) => (
                  <label key={l} className={cx(styles.langKey, l === lang && styles.keyActive)}>
                    <input
                      type="radio"
                      className="srOnly"
                      name="lang"
                      value={l}
                      checked={l === lang}
                      onChange={() => onLangChange(l)}
                    />
                    <span lang={l}>{l.toUpperCase()}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            <ToggleSwitch label={t.lamp} checked={lampOn} onChange={onLampChange} withLed={false} />
          </div>

          <nav className={styles.selector} aria-label={t.sections}>
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) => cx(styles.key, isActive && styles.keyActive)}
              >
                {({ isActive }) => (
                  <>
                    <Led state={isActive ? 'on' : 'off'} />
                    <span>{item.label}</span>
                  </>
                )}
              </NavLink>
            ))}
          </nav>
        </header>

        <main className={styles.main}>{children}</main>
      </div>
      <div className={styles.side} aria-hidden />
    </div>
  );
}
