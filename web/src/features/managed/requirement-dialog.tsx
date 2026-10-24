"use client";
import { useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { readPage } from "@/lib/api";
import type { MutationResult } from "@/lib/types";
import { RequirementForm } from "../automation/requirement-form";
import { useProjectScope } from "../projects/project-context";

export function RequirementDialog({
  prompt,
  onClose,
  onSuccess,
}: {
  prompt: string;
  onClose: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const scope = useProjectScope();
  const locked = useRef(false);
  // The request checks the captured project ID; the prompt stays in this
  // mounted dialog and never enters a URL, browser storage or shared cache.
  const result = useQuery({
    queryKey: ["managed-requirement", scope.key],
    queryFn: ({ signal }) =>
      readPage(scope.app("/app/automation"), signal, scope.key),
    retry: false,
    gcTime: 0,
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !locked.current) onClose();
      }}
    >
      <DialogContent
        className="managed-dialog"
        aria-describedby="managed-requirement-help"
      >
        <DialogHeader>
          <DialogTitle>Выполнить требование</DialogTitle>
        </DialogHeader>
        <p id="managed-requirement-help" className="form-help">
          Исходное поручение перенесено в форму. Выберите действующее разрешение
          команды и подтвердите отправку. Аналитик создаст задачи, затем команда
          выполнит разработку, независимое ревью и проверку.
        </p>
        {result.isPending ? (
          <p className="form-help" role="status">
            Загружаем разрешения проекта…
          </p>
        ) : result.isError ? (
          <div className="screen-stack">
            <p className="notice error-notice" role="alert">
              Не удалось загрузить форму. Исходное поручение остаётся в этой
              работе.
            </p>
            <button
              className="button button-secondary"
              onClick={() => void result.refetch()}
            >
              Повторить загрузку
            </button>
          </div>
        ) : (
          <RequirementForm
            page={result.data}
            initialText={prompt}
            onLockChange={(value) => {
              locked.current = value;
            }}
            onSuccess={(response) => {
              onClose();
              onSuccess(response);
            }}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
