package worker

import (
	"errors"
	"fmt"
)

// launchCause is a fixed, owner-readable reason an automation attempt did not
// start. Code is stable for UI/diagnostics; the English and Russian texts never
// embed paths, tokens or raw OS errors (only the workspace alias and counts).
type launchCause struct{ code, en, ru string }

func (c *launchCause) Error() string { return c.code + ": " + c.en + " / " + c.ru }

func cause(code, en, ru string) error { return &launchCause{code, en, ru} }

func dirtyWorkspaceCause(alias string, files int, sameStep bool) error {
	if sameStep {
		return cause("workspace_dirty", fmt.Sprintf("workspace %s has %d changed files left by this same interrupted step; a dirty resume is not supported, cancel the step or commit/stash the files", alias, files),
			fmt.Sprintf("в рабочей папке %s %d изменённых файлов этого же прерванного шага; продолжение с грязным деревом не поддерживается — отмените шаг или закоммитьте/уберите в stash", alias, files))
	}
	return cause("workspace_dirty", fmt.Sprintf("workspace %s has %d uncommitted or untracked files that this worker cannot attribute to an interrupted step; commit, stash or remove them", alias, files),
		fmt.Sprintf("в рабочей папке %s %d незакоммиченных или неотслеживаемых файлов, которые worker не может отнести к прерванному шагу; закоммитьте, уберите в stash или удалите их", alias, files))
}

func filteredWorkspaceCause(alias string, files int) error {
	return cause("workspace_dirty", fmt.Sprintf("workspace %s has %d changed files with a Git filter attribute (.gitattributes filter=); the worker does not run filters to stash them, commit or stash them yourself", alias, files),
		fmt.Sprintf("в рабочей папке %s %d изменённых файлов с Git-фильтром (.gitattributes filter=); worker не запускает фильтры для stash — закоммитьте или уберите их в stash сами", alias, files))
}

var (
	errTargetSHA = cause("target_sha_mismatch", "reviewed full SHA differs from local HEAD", "проверяемый полный SHA не совпадает с локальным HEAD")
	errProofOpen = cause("previous_proof_unresolved", "previous automation exit proof of this run is not acknowledged yet", "прежнее доказательство завершения этого запуска ещё не подтверждено сервером")
	errBusy      = cause("workspace_busy", "the checkout is owned by another local worker or has an unresolved previous launch; see `agent-board worker recover`", "checkout занят другим локальным worker или имеет незавершённый прежний запуск; см. `agent-board worker recover`")
	errSource    = cause("source_proof_unavailable", "Git source proof of the workspace failed (index differs from HEAD, unsupported entry or Git unavailable)", "не удалось получить Git-доказательство рабочей папки (индекс отличается от HEAD, неподдерживаемый файл или Git недоступен)")
	errIPC       = cause("ipc_unavailable", "local verification IPC endpoint could not be created", "не удалось создать локальный IPC-канал проверок")
	errStart     = cause("native_start_failed", "native provider could not start; check its local installation and permissions", "не удалось запустить native provider; проверьте его локальную установку и права")
	errStopped   = cause("stopped_during_preparation", "attempt was stopped, revoked or its deadline passed during local preparation", "попытка остановлена, отозвана или истёк срок во время локальной подготовки")
	errConsent   = cause("local_consent_unavailable", "local automation consent does not allow this role, provider, model or deadline", "локальное согласие на автоматизацию не разрешает эту роль, provider, модель или срок")
)

func budgetCause(detail string) error {
	return cause("local_budget_exhausted", "local workflow budget exhausted ("+detail+"); raise the local limit with explicit consent or replan", "локальный бюджет workflow исчерпан ("+detail+"); увеличьте локальный предел явным согласием или перепланируйте")
}

// registrationCause keeps the bounded server status/code/message, never a body.
func registrationCause(err error) error {
	var status statusError
	if errors.As(err, &status) {
		return cause("registration_rejected", "scoped execution registration rejected ("+status.Error()+")", "сервер отклонил регистрацию scoped execution")
	}
	return cause("registration_unavailable", "scoped execution registration did not complete (network or timeout)", "регистрация scoped execution не завершилась (сеть или таймаут)")
}

// classifyPreparation maps an internal error to its stable cause. Errors that
// already carry a cause keep it; unknown errors keep a generic fixed text.
func classifyPreparation(err error) (code, message string) {
	var lc *launchCause
	var pin pinError
	switch {
	case errors.As(err, &lc):
		return lc.code, lc.Error()
	case errors.As(err, &pin):
		return pin.Code(), pin.Code() + ": " + pin.Error()
	case errors.Is(err, errPermissionProof):
		return "native_git_permission_unproven", errPermissionProof.Error()
	}
	return "preparation_failed", "preparation_failed: automation local consent, source proof or scoped registration is unavailable / локальное согласие, доказательство исходников или регистрация недоступны"
}
