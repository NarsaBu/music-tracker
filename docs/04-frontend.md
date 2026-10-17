# 04 — Frontend

## 1. Стек

- React 19 + TypeScript (strict), Vite
- React Router — маршрутизация
- TanStack Query — кэш серверного состояния (используется внутри слоя Model)
- Zod — валидация ответов API на границе
- CSS Modules + CSS custom properties (дизайн-токены) — см. [10-design.md](10-design.md)
- Storybook — UI-кит
- Vitest + React Testing Library, MSW для моков API

## 2. MVC в React

| Слой | Что это в проекте | Знает о… | Не знает о… |
|------|-------------------|----------|-------------|
| **Model** | Типы сущностей, API-клиенты (repositories), query/mutation-хуки данных, чистые функции бизнес-логики | HTTP, схемах данных | React-компонентах, разметке |
| **Controller** | Хуки `useXxxController`: связывают модели, держат UI-состояние (фильтры, выбор, шаги), обрабатывают события, возвращают готовый view-model | Model | Разметке, CSS |
| **View** | Компоненты, получающие всё через props; без запросов и бизнес-логики | Только props | API, моделях |

Страница (`*Page.tsx`) — тонкая склейка: вызывает контроллер и передаёт результат во View.

```tsx
// features/subscriptions/SubscriptionsPage.tsx
export function SubscriptionsPage() {
  const vm = useSubscriptionsController();
  return <SubscriptionsView {...vm} />;
}
```

```ts
// features/subscriptions/controller/useSubscriptionsController.ts
export function useSubscriptionsController() {
  const [search, setSearch] = useState('');
  const { data, isLoading } = useSubscriptionsQuery();      // Model
  const unsubscribe = useUnsubscribeMutation();              // Model
  const updateFilter = useUpdateSubscriptionMutation();      // Model
  const items = useMemo(() => filterByName(data ?? [], search), [data, search]);
  return {
    items, isLoading, search,
    onSearchChange: setSearch,
    onUnsubscribe: (mbid: string) => unsubscribe.mutate(mbid),
    onToggleFeatured: (mbid: string, value: boolean | null) =>
      updateFilter.mutate({ mbid, include_featured: value }),
  };
}
```

### SOLID / DRY на фронте
- *SRP*: компонент View отвечает только за отображение; контроллер — за одну страницу/виджет.
- *OCP*: переиспользуемые компоненты расширяются через props/composition (`renderItem`, `slots`), а не флагами-переключателями.
- *DIP*: модели зависят от интерфейса `HttpClient`, а не от `fetch` напрямую — подменяется в тестах.
- *DRY*: общий `HttpClient` (базовый URL, маппинг ошибок), UI-кит в `shared/ui`, дизайн-токены, общие query keys, общий компонент `FilterOverrideEditor` (используется и в глобальных настройках, и в настройках подписки).

## 3. Структура

```
frontend/
├── src/
│   ├── app/                        # bootstrap: роутер, провайдеры (QueryClient), layout «стойки»
│   ├── shared/
│   │   ├── api/                    # HttpClient, ApiError
│   │   ├── i18n/                   # словари RU/EN, I18nProvider, useI18n (язык — переключатель в шапке)
│   │   ├── ui/                     # UI-кит (View): Panel, ToggleSwitch, RecordSleeve, …
│   │   │   └── theme/              # tokens.css, textures.svg, fonts.css
│   │   ├── lib/                    # утилиты (частичные даты MusicBrainz и т.п.)
│   │   └── config/
│   ├── entities/                   # Model: общие сущности
│   │   ├── artist/      { model.ts, schema.ts, api.ts, queries.ts }
│   │   ├── release/
│   │   ├── subscription/
│   │   └── settings/
│   └── features/                   # фичи = Controller + View
│       ├── artist-search/          # поиск и подписка (US-1)
│       ├── subscriptions/          # список подписок, override фильтров и featuring (US-4, US-5, US-6)
│       ├── import/                 # загрузка файла, прогресс, ручное разрешение (US-2, US-3)
│       ├── releases-feed/          # лента новинок (US-9)
│       ├── sync/                   # кнопка ручного синка + VU-метр прогресса (US-10)
│       └── settings/               # фильтры, дайджест, email, RSS (US-5, US-7, US-8)
│           ├── SettingsPage.tsx
│           ├── controller/useSettingsController.ts
│           └── view/SettingsView.tsx
└── .storybook/
```

## 4. Экраны

Визуальные метафоры экранов — в [10-design.md §1](10-design.md#1-концепция--hi-fi-стойка).

| Маршрут | Экран | Основные элементы |
|---------|-------|-------------------|
| `/` | Лента новинок | конверты пластинок: обложка, исполнитель, название, тип, дата, бейдж `feat.`; фильтр по типу и роли |
| `/artists` | Подписки | карточки исполнителей, поиск, отписка, «Проверить сейчас», override типов и тумблер featuring |
| `/artists/add` | Поиск исполнителя | поле ввода с debounce 400 мс, кандидаты (имя, disambiguation, страна, годы, score), тумблер featuring, кнопка «Подписаться» |
| `/artists/:mbid` | Исполнитель | каталог релиз-групп (вкл. featured), настройки фильтра |
| `/imports` | Импорт | drag&drop файла, список импортов с прогрессом |
| `/imports/:id` | Разбор импорта | строки `needs_review` с выбором кандидата / «пропустить» |
| `/settings` | Настройки | тумблеры primary/secondary types, featuring по умолчанию, анонсы; email (адрес, тест), дайджест (daily/weekly, день, время, часовой пояс); RSS-ссылка + «перевыпустить» |

## 5. Соглашения

- ESLint (typescript-eslint, react-hooks) + Prettier + Stylelint.
- Контроллеры тестируются через `renderHook` с MSW; View — RTL и Storybook без моков сети.
- Ответы API валидируются Zod-схемой в `entities/*/api.ts`; наружу отдаются доменные типы.
- Никаких `fetch` в компонентах и контроллерах — только через Model.
- Никаких цветов/теней вне токенов.
