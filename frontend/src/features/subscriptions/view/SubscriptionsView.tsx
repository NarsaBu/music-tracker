import { useRef, type ReactNode } from 'react';
import { Link } from 'react-router';
import { artistTypeLabel } from '@/entities/artist/model';
import { useI18n } from '@/shared/i18n';
import { LCD_BLANK, lcdCount } from '@/shared/lib/format';
import {
  EngravedInput,
  EngravedSelect,
  IndexCard,
  LcdDisplay,
  Led,
  Pager,
  Panel,
  PushButton,
} from '@/shared/ui';
import { PAGE_SIZES, type SubscriptionsViewModel } from '../controller/useSubscriptionsController';
import styles from './SubscriptionsView.module.css';

export interface SubscriptionsViewProps extends SubscriptionsViewModel {
  /** Slot above the drawer, e.g. the sync control. */
  header?: ReactNode;
}

/** Countries by their localized names ("Великобритания"), sorted; unknown codes stay as is. */
function countryOptions(codes: readonly string[], locale: string) {
  let names: Intl.DisplayNames | undefined;
  try {
    names = new Intl.DisplayNames([locale], { type: 'region' });
  } catch {
    // no region names in this runtime: show the codes
  }
  const nameOf = (code: string) => {
    try {
      return names?.of(code) ?? code;
    } catch {
      return code;
    }
  };
  return codes
    .map((code) => ({ code, name: nameOf(code) }))
    .sort((a, b) => a.name.localeCompare(b.name, locale))
    .map(({ code, name }) => ({ value: code, label: name }));
}

export function SubscriptionsView({ header, ...vm }: SubscriptionsViewProps) {
  const { t: all } = useI18n();
  const t = all.subscriptions;
  const listTop = useRef<HTMLDivElement>(null);

  const typeOptions = [
    { value: '', label: t.anyType },
    ...vm.facets.types.map((type) => ({
      value: type,
      label: artistTypeLabel(type, all.common) ?? type,
    })),
  ];
  const countries = [
    { value: '', label: t.anyCountry },
    ...countryOptions(vm.facets.countries, all.locale),
  ];

  const pager = (bottom: boolean) => (
    <Pager
      page={vm.page}
      pageCount={vm.pageCount}
      label={t.pages}
      onPageChange={(p) => {
        vm.onPageChange(p);
        // from the bottom of a long drawer, go back to its first card
        if (bottom) listTop.current?.scrollIntoView({ block: 'start', behavior: 'smooth' });
      }}
    />
  );

  return (
    <div className={styles.layout}>
      {header}
      <Panel
        title={t.title}
        extra={
          <LcdDisplay label={t.total} size="sm">
            {vm.isLoading ? LCD_BLANK : lcdCount(vm.total)}
          </LcdDisplay>
        }
      >
        <div className={styles.toolbar}>
          <EngravedInput
            className={styles.search}
            label={t.search}
            type="search"
            autoComplete="off"
            value={vm.search}
            onChange={(e) => vm.onSearchChange(e.target.value)}
          />
          <PushButton variant="primary" onClick={vm.onAdd}>
            {t.add}
          </PushButton>
        </div>

        <div className={styles.filters}>
          <EngravedSelect
            label={t.filterStatus}
            value={vm.filters.status}
            onChange={(v) => vm.onFilterChange('status', v)}
            options={[
              { value: '', label: t.statusOptions.all },
              { value: 'synced', label: t.statusOptions.synced },
              { value: 'pending', label: t.statusOptions.pending },
            ]}
          />
          {/* a filter with nothing to choose from is not shown */}
          {typeOptions.length > 1 && (
            <EngravedSelect
              label={t.filterType}
              value={vm.filters.type}
              onChange={(v) => vm.onFilterChange('type', v)}
              options={typeOptions}
            />
          )}
          {countries.length > 1 && (
            <EngravedSelect
              label={t.filterCountry}
              value={vm.filters.country}
              onChange={(v) => vm.onFilterChange('country', v)}
              options={countries}
            />
          )}
          <EngravedSelect
            label={t.filterRule}
            value={vm.filters.filter}
            onChange={(v) => vm.onFilterChange('filter', v)}
            options={[
              { value: '', label: t.ruleOptions.all },
              { value: 'own', label: t.ruleOptions.own },
              { value: 'global', label: t.ruleOptions.global },
            ]}
          />
          <EngravedSelect
            label={t.sort}
            value={vm.filters.sort}
            onChange={(v) => vm.onFilterChange('sort', v)}
            options={[
              { value: 'name', label: t.sortOptions.name },
              { value: 'added', label: t.sortOptions.added },
            ]}
          />
          <EngravedSelect
            label={t.pageSize}
            value={String(vm.pageSize)}
            onChange={(v) => vm.onPageSizeChange(Number(v))}
            options={PAGE_SIZES.map((n) => ({ value: String(n), label: String(n) }))}
          />
          {(vm.filtered || vm.search) && (
            <PushButton className={styles.reset} onClick={vm.onResetFilters}>
              {t.resetFilters}
            </PushButton>
          )}
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          {all.common.serverUnreachable}
        </p>
      )}

      {vm.isEmpty && <p className={styles.message}>{t.empty}</p>}

      {!vm.isEmpty && !vm.isLoading && vm.cards.length === 0 && !vm.isError && (
        <p className={styles.message}>
          {vm.search ? t.nothingFound(vm.search) : t.nothingFiltered}
        </p>
      )}

      {vm.cards.length > 0 && (
        <>
          <div ref={listTop} className={styles.listHead}>
            <span className={styles.range} aria-live="polite">
              {t.shown(vm.range.from, vm.range.to, vm.total)}
            </span>
            {pager(false)}
          </div>
          <ul className={styles.drawer} aria-label={t.list}>
            {vm.cards.map((c) => (
              <li key={c.mbid}>
                <IndexCard
                  title={
                    <Link className={styles.cardLink} to={`/artists/${c.mbid}`}>
                      {c.name}
                    </Link>
                  }
                  tab={c.country ?? undefined}
                  tiltSeed={c.mbid}
                  extra={<Led state={c.synced ? 'ok' : 'warn'} label={c.syncText} />}
                  footer={
                    <PushButton
                      disabled={c.unsubscribing}
                      onClick={() => vm.onUnsubscribe(c.mbid)}
                      aria-label={t.unsubscribeFrom(c.name)}
                    >
                      {t.unsubscribe}
                    </PushButton>
                  }
                >
                  {c.details && <span className={styles.details}>{c.details}</span>}
                  <dl className={styles.facts}>
                    <dt>{t.types}</dt>
                    <dd>
                      {c.typesText} <small>({t.source[c.typesSource]})</small>
                    </dd>
                    <dt>{all.common.featuring}</dt>
                    <dd>
                      {c.featuring ? all.common.yes : all.common.no}{' '}
                      <small>({t.source[c.featuringSource]})</small>
                    </dd>
                    <dt>{t.status}</dt>
                    <dd>{c.syncText}</dd>
                  </dl>
                </IndexCard>
              </li>
            ))}
          </ul>
          {pager(true)}
        </>
      )}
    </div>
  );
}
