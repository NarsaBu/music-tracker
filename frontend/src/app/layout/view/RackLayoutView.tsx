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
  navItems,
  statusText,
  statusLed,
  lampOn,
  onLampChange,
  children,
}: RackLayoutViewProps) {
  return (
    <div className={styles.rack}>
      <div className={styles.side} aria-hidden />
      <div className={styles.column}>
        <header className={styles.receiver}>
          <div className={styles.brand}>
            <span className={styles.brandName}>Music Tracker</span>
            <span className={styles.brandModel}>Model MT-1 · Release monitor</span>
          </div>

          <div className={styles.status}>
            <Led state={statusLed} />
            <LcdDisplay label="Состояние сервера" size="sm">
              {statusText}
            </LcdDisplay>
          </div>

          <ToggleSwitch
            className={styles.lamp}
            label="Lamp"
            checked={lampOn}
            onChange={onLampChange}
            withLed={false}
          />

          <nav className={styles.selector} aria-label="Разделы">
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
