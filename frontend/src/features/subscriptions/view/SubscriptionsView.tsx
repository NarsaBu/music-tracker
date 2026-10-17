import { EngravedInput, IndexCard, LcdDisplay, Led, Panel, PushButton } from '@/shared/ui';
import type { SubscriptionsViewModel } from '../controller/useSubscriptionsController';
import styles from './SubscriptionsView.module.css';

const SOURCE_TEXT = { own: 'своё', global: 'общее' } as const;

export function SubscriptionsView(vm: SubscriptionsViewModel) {
  return (
    <div className={styles.layout}>
      <Panel
        title="Картотека исполнителей"
        extra={
          <LcdDisplay label="Всего подписок" size="sm">
            {vm.isLoading ? '---' : String(vm.total).padStart(3, '0')}
          </LcdDisplay>
        }
      >
        <div className={styles.toolbar}>
          <EngravedInput
            className={styles.search}
            label="Найти в картотеке"
            type="search"
            autoComplete="off"
            value={vm.search}
            onChange={(e) => vm.onSearchChange(e.target.value)}
          />
          <PushButton variant="primary" onClick={vm.onAdd}>
            Добавить исполнителя
          </PushButton>
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          Не удалось связаться с сервером
        </p>
      )}

      {vm.isEmpty && (
        <p className={styles.message}>
          Ящик пуст. Найдите исполнителя на тюнере и подпишитесь — карточка появится здесь.
        </p>
      )}

      {!vm.isEmpty && !vm.isLoading && vm.cards.length === 0 && !vm.isError && (
        <p className={styles.message}>Никого не нашлось по запросу «{vm.search}».</p>
      )}

      {vm.cards.length > 0 && (
        <ul className={styles.drawer} aria-label="Подписки">
          {vm.cards.map((c) => (
            <li key={c.mbid}>
              <IndexCard
                title={c.name}
                tab={c.country ?? undefined}
                tiltSeed={c.mbid}
                extra={<Led state={c.synced ? 'ok' : 'warn'} label={c.syncText} />}
                footer={
                  <PushButton
                    disabled={c.unsubscribing}
                    onClick={() => vm.onUnsubscribe(c.mbid)}
                    aria-label={`Отписаться от ${c.name}`}
                  >
                    Отписаться
                  </PushButton>
                }
              >
                {c.details && <span className={styles.details}>{c.details}</span>}
                <dl className={styles.facts}>
                  <dt>Типы</dt>
                  <dd>
                    {c.typesText} <small>({SOURCE_TEXT[c.typesSource]})</small>
                  </dd>
                  <dt>Featuring</dt>
                  <dd>
                    {c.featuring ? 'да' : 'нет'} <small>({SOURCE_TEXT[c.featuringSource]})</small>
                  </dd>
                  <dt>Статус</dt>
                  <dd>{c.syncText}</dd>
                </dl>
              </IndexCard>
            </li>
          ))}
        </ul>
      )}

      {vm.pageCount > 1 && (
        <nav className={styles.pager} aria-label="Страницы картотеки">
          <PushButton onClick={vm.onPrevPage} disabled={vm.page === 0}>
            ← Назад
          </PushButton>
          <LcdDisplay label="Страница" size="sm">
            {`${vm.page + 1}/${vm.pageCount}`}
          </LcdDisplay>
          <PushButton onClick={vm.onNextPage} disabled={vm.page >= vm.pageCount - 1}>
            Вперёд →
          </PushButton>
        </nav>
      )}
    </div>
  );
}
