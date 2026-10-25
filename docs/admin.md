# Администрирование: люди, токены, роли

Контракт от 25.10.2026 по прямому поручению пользователя: управлять людьми,
выдавать и отзывать токены из браузера, иметь ролевую модель с глобальной
областью. Расширяет [multi-project.md](multi-project.md) и меняет SEC-06 в
[security-contract.md](security-contract.md): выдача токенов из браузера
разрешена только глобальному администратору, секрет показывается один раз.

## Модель

- **Человек** (`people`) — учётная сущность: имя, заметка, глобальная роль
  (`member` | `admin`), активность, версия. ID `u_<32hex>`.
- **Identity** — как и раньше, один credential (`legacy_<tokenID>` или
  `i_<32hex>`), привязанный к owner/machine. Новое поле `person_id` связывает
  credential с человеком. У одного человека может быть несколько identity
  (разные компьютеры). Identity без человека допустимы (исторические), админ
  может привязать их.
- **Роли проекта** не меняются: owner / editor / viewer в `memberships`.
- **Глобальная роль `admin`** (у человека): управление людьми, выдача и отзыв
  credentials, привязка identity, глобальные роли, просмотр и управление
  участниками **любого** проекта (`can_manage=true` для всех проектов).
  Запись на доску проекта по-прежнему требует членства editor/owner — админ
  может выдать его себе. Админ **не** получает control чужого ПК, host-права и
  OS-управление: ось managed/automation не меняется.
- **Деактивация человека** (`active=0`): все его identity перестают
  проходить `identity()`; браузерные сессии и native-запросы отвергаются,
  credentials не отзываются, повторная активация возвращает доступ.
- Защита: нельзя деактивировать или понизить **последнего активного
  админа**; нельзя деактивировать себя.

## Хранилище: registry schema 2

Дополнительная миграция registry.sqlite 1 → 2 при открытии (и offline CLI):

```sql
CREATE TABLE IF NOT EXISTS people(id TEXT PRIMARY KEY, name TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '', role TEXT NOT NULL DEFAULT 'member'
  CHECK(role IN ('member','admin')), active INTEGER NOT NULL DEFAULT 1,
  version INTEGER NOT NULL DEFAULT 1, created INTEGER NOT NULL, updated INTEGER NOT NULL);
ALTER TABLE identities ADD COLUMN person_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS registry_identity_person ON identities(person_id);
CREATE TABLE IF NOT EXISTS admin_audit(id INTEGER PRIMARY KEY, event TEXT NOT NULL,
  actor_id TEXT NOT NULL, person_id TEXT NOT NULL DEFAULT '',
  identity_id TEXT NOT NULL DEFAULT '', details TEXT NOT NULL DEFAULT '', at INTEGER NOT NULL);
PRAGMA user_version=2;
```

Перед DDL делается consistent копия registry (как у board `.migrations`):
`<projects-data>/registry.sqlite.migrations/registry1-<sha256>.db`, каталог
0700, файл 0600. Миграция идёт в одной транзакции; при любой ошибке registry
остаётся schema 1.

Bootstrap внутри миграции: для **каждой** существующей identity создаётся
свой человек (имя = owner, при совпадении имён добавляется machine), identity
привязывается; человек получает `admin`, если его identity — активный
`owner` **default-проекта**. Объединять людей по owner-метке миграция не
имеет права (метки не ключ); админ объединяет вручную через привязку.
`importLegacy` и `projects token-create` без `--person` тоже создают человека.

Capabilities: `registry_schema: [2]`, новый постоянный floor
`people_admin: 1`; guard требует точное равенство, как раньше.
Прежний binary (schema 1) отвергается после миграции.

## CLI (offline, как `bootstrap-owner`)

- `projects bootstrap-admin --identity ID` — создать человека при
  необходимости и назначить `admin` (восстановление доступа без браузера).
- `projects people-list` — JSON людей и их identity без секретов.
- `projects token-create ... [--person u_...]` — привязать новый credential к
  существующему человеку.

## Страница `GET /web/catalog/admin` (только admin, иначе 404)

```json
{"ok":true,"page":{"name":"admin",
  "registry":{"identity":{"id","owner","machine","admin":true},"csrf":"…","actions":{…}},
  "people":[{"id","name","note","role","active","version","created_at","updated_at"}],
  "identities":[{"id","person_id","owner","machine","source":"legacy|catalog","expires_at","revoked"}],
  "projects":[{"id","name","state"}],
  "memberships":[{"project_id","identity_id","role"}],
  "audit":[{"id","event","actor_id","person_id","identity_id","details","at"}]}}
```

`registry.identity.admin` отдаётся на **всех** catalog-страницах (`/web/projects`,
`/web/catalog/projects/{id}`), чтобы клиент показывал пункт меню.

Ключи actions (выдаются только админу): `admin:person:create`,
`admin:person:{id}:update`, `admin:person:{id}:credential`,
`admin:identity:{id}:revoke`, `admin:identity:{id}:link`.
`audit` — последние 200 записей, новые первыми.

## Мутации (POST, `csrf` + `action_id`, идемпотентны как catalog actions)

| путь | поля | результат |
|------|------|-----------|
| `/web/catalog/admin/people` | `name` (1–64), `note` (≤512), `role` | `{"ok":true,"person":{…}}` |
| `/web/catalog/admin/people/{id}` | `action=update\|deactivate\|activate`, `name`, `note`, `role`, `version` | `{"ok":true,"person":{…}}` |
| `/web/catalog/admin/people/{id}/credentials` | `machine` (1–64), `days` (1–365) | `{"ok":true,"issued":{"identity_id","owner","machine","expires_at","secret"}}` |
| `/web/catalog/admin/identities/{id}` | `action=revoke\|link`, `person_id` | `{"ok":true,"identity":{…}}` |

Правила секрета: `secret` присутствует **только** в живом ответе на первый
успешный POST. В `actions.result` сохраняется результат без `secret` и с
`"secret_delivered":true`; replay возвращает его без секрета. Секрет не
попадает в URL, redirect, draft, журнал, audit. Ответ — `Cache-Control: no-store`.
Owner нового credential = имя человека; `expires_at` ≤ 365 дней.
Отзыв catalog-credential — `RevokeCredential`; legacy — `RevokeToken` на
default store; оба пишут `admin_audit`.
Все мутации пишут `admin_audit` с `actor_id` (identity админа).
Ошибки — через существующий `fail()`: `invalid`, `denied`, `not_found`, `conflict`.

Версии: `people.version` растёт при каждом изменении; `update` с устаревшей
`version` → `conflict`.

## Клиент

- Маршрут `/app/admin` ↔ `/web/catalog/admin` (scope `catalog`), пункт
  «Администрирование» в группе «Организация», виден только `admin`.
- Экран: список людей (имя, роль, статус, число identity, проекты), карточка
  человека (имя/заметка/роль/активность), его identity (machine, источник,
  срок, отозван, кнопка отзыва), форма «Выдать токен» (machine, срок) с
  показом секрета один раз: копирование и скачивание файла
  `<owner>-<machine>.token`; привязка непривязанных identity; журнал.
- POST-пути `/web/catalog/admin/...` добавляются в allowlist `submitAction`.

## Что не входит

Управление hosts/controllers (остаётся native CLI), OAuth/GitHub-вход,
групповые/кастомные роли, per-permission матрица. Глобальная роль одна:
`admin`; остальные права выражаются ролями проектов.
