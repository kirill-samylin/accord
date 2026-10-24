# WORK255 — LaunchAgent startup: bounded read-only source diagnostic

24.10.2026. QA scope только source inspection и эта карточка. Exact inspected
revision `be95daed7153f4abeb72012286cc748d0596b479`, не текущий mutable source tree.
Binary installation/hash и live epoch/OS/process/network observations — evidence
root WORK251, не повторялись этим QA. Никакие worker/helper/provider/launchctl/
HTTP/process/config probes не запускались; user/global/GUI config, credentials,
proof/state/token/env не читались. Plist onboarding224V5 — source template only.

## Установлено по source

- cmd/agent-board/worker.go:87 печатает «worker is running» ПЕРЕД engine.Run.
  Это banner успешного Open, не network-readiness/ownership/capabilities proof.
- runner.go:18 сначала probeProviders; только затем newRemote, session open,
  reconcile, managed/poll. Host capabilities отправляются в pollInput. Epoch0 и
  отсутствующие capabilities совместимы с остановкой до первого poll, но не
  локализуют точный вызов. Session open/reconcile тоже до него.
- probe.go: для каждого configured provider последовательно --version и help;
  для Codex второй вызов exec --help. Это preflight, не model turn.
- discovery.go: readNativeFor вызывает starter=startProcess ДО создания
  5s timer. process_unix.go: cmd.Start синхронен. Если Start задержался, timer ещё
  не создан. Поэтому wall time9min не исключён этим 5s source guard; actual
  stack/spawn proof отсутствует, зависание Start НЕ доказано.
- Native preflight stderr направлен в io.Discard, поэтому пустой worker stderr
  не доказывает отсутствие native diagnostics. После запуска чтениеstdout,
  cleanup/reap/Wait тоже может предшествовать возврату probe.
- providerEnv наследует os.Environ, удаляя только AGENT_BOARD_*/ACCORD_*;
  HOME/TMPDIR/USER/LOGNAME/PATH не создаются и не нормализуются. Одинаковый
  наследуемый принцип используется в native Start и worker self-supervisor.
  Начальный probe использует direct startProcess, НЕ worker-supervise helper.
- Config/binary/provider/checkout paths абсолютные; PATH не выбирает provider
  executable. Добавление node/git/toolchain paths не обосновано первым poll.
- workspace_lock.go использует os.UserConfigDir только при acquireWorkspace
  для task/run. Darwin implementation в установленном Go source требует HOME.
  Это реальная необходимость HOME для будущей работы, но не объяснение epoch0:
  local workspace-lock ещё не должен требоваться до первой dispatch.
- В Go worker source до firstpoll нет прямого TMPDIR/USER/LOGNAME requirement.
  Go tempDir при отсутствующем TMPDIR имеет /tmp fallback. Какие переменные
  требует сам установленный provider на этой машине, этим read-only scope
  не проверено. Нет доказательства, что HOME отсутствовал в реальном job env:
  root наблюдал отсутствие только в launchd-global getenv, это другой слой.

## Минимальная рекомендация root (не применена)

Сначала рассматривать explicit HOME=current account real home при сохранении
текущего PATH и всех остальных plist keys/WorkingDirectory/ProcessType/binary/
workerconfig/state/consent. Это устраняет неоднозначность inherited home без
смены provider CODEX_HOME/auth/trust, но пока гипотеза, не исправленная причина.
Однопеременный вариант лучше различает эффект; если root выбирает стандартный
пакет env, безопасные имена HOME, USER, LOGNAME, PATH, TMPDIR, значения только
своего текущего пользователя. USER/LOGNAME — parity, не доказанная worker
необходимость; PATH оставить source-template system PATH; TMPDIR добавлять лишь
root-verified existing canonical per-user directory, не выдумывать /var/folders
suffix и не создавать/копировать чужой temp. CODEX_HOME не переносить из GUI и
не менять ради опыта. Proxy/cert/global settings не менять.

GUI reference использован только как имена из source plist/documentation:
HOME/CODEX_HOME/USER/LOGNAME/PATH; никаких его values/config/state не извлекалось.
Это accepted working configuration precedent, не причинное доказательство.

ProcessType=Background также отличается от foreground execution context;
privacy/responsibility/scheduler/spawn — альтернативы, не доказанные диагнозы.
Не менять ProcessType одновременно с HOME, не расширять sandbox/permissions и
не патчить worker, пока конкретная first-cause не установлена. Root-owned
следующая разрешённая проверка должна отдельно показать first successful
poll/epoch/capabilities и ownership/cleanup; banner/root-only tree/API200 от
другого диагностического клиента не заменяют её. Запуск выполняет только root
по отдельному operation gate, не эта QA-карточка.

## Результат и ограничения

**SOURCE REVIEW COMPLETE / actual cause UNKNOWN.** Root-reported foreground
success с тем же binary/config подтверждает контекстное различие, не HOME/TCC/
ProcessType cause. Нет основания менять backend auth/lease/state/config/proof
или permissions. Возможный отдельный future robustness scope — честная startup
readiness и ограничение синхронного startup до timer; это не сделанный fix и не
разрешение на product patch. Runtime/OS диагноз, env-only trial и устойчивый
LaunchAgent firstpoll: **NOT-RUN этим QA**.
