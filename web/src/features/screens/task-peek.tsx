"use client";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Dialog as Primitive } from "radix-ui";
import { X } from "lucide-react";
import type { Task, MutationResult } from "@/lib/types";
import { readPage } from "@/lib/api";
import { useDialogFocusHandlers } from "@/components/ui/dialog-focus";
import { TaskDetail } from "./task-detail";
export function TaskPeek({
  task,
  onClose,
  onSuccess,
}: {
  task: Task | null;
  onClose: () => void;
  onSuccess: (result: MutationResult) => void;
}) {
  const focus = useDialogFocusHandlers();
  const result = useQuery({
    queryKey: ["peek", task?.key],
    queryFn: ({ signal }) => readPage(`/app/tasks/${task!.key}`, signal),
    enabled: Boolean(task),
    retry: false,
    gcTime: 0,
  });
  return (
    <Primitive.Root
      open={Boolean(task)}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <Primitive.Portal>
        <Primitive.Overlay className="dialog-overlay" />
        <Primitive.Content
          className="task-peek"
          aria-describedby={undefined}
          {...focus}
        >
          <Primitive.Title className="sr-only">
            Быстрый просмотр {task?.key}
          </Primitive.Title>
          <div className="peek-top">
            <Link
              prefetch={false}
              className="inline-link mono"
              href={`/app/tasks/${task?.key}`}
              onClick={onClose}
            >
              Открыть {task?.key}
            </Link>
            <Primitive.Close
              className="icon-button"
              aria-label="Закрыть просмотр"
            >
              <X size={18} />
            </Primitive.Close>
          </div>
          {result.isPending ? (
            <div className="page">
              <div className="skeleton" />
              <div className="skeleton" />
            </div>
          ) : result.isError ? (
            <div className="page">
              <div className="notice error-notice" role="alert">
                Не удалось открыть задачу
              </div>
              <button
                className="button button-secondary"
                onClick={() => result.refetch()}
              >
                Повторить
              </button>
            </div>
          ) : result.data ? (
            <TaskDetail
              key={result.data.task?.key}
              page={result.data}
              onSuccess={(result) => {
                onClose();
                onSuccess(result);
              }}
              compact
            />
          ) : null}
        </Primitive.Content>
      </Primitive.Portal>
    </Primitive.Root>
  );
}
