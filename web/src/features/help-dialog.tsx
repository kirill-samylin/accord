"use client";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function HelpDialog({
  open,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent aria-describedby="help-steps">
        <DialogHeader>
          <DialogTitle>Как начать работу</DialogTitle>
        </DialogHeader>
        <ol id="help-steps" className="help-steps">
          <li>
            <strong>Эпик</strong> — крупная цель проекта, например «Запуск
            продукта». Каждая задача входит в эпик.
          </li>
          <li>
            <strong>Задача</strong> — конкретная работа с описанием результата.
            Роль исполнителя решает, кто её возьмёт: агент с этой ролью или
            человек (тогда задача попадёт в «Вопросы»).
          </li>
          <li>
            <strong>Приоритет</strong> 0–100: задачи с меньшим числом берут в
            работу раньше.
          </li>
          <li>
            Агенты подключаются к проекту отдельно; на «Обзоре» видно, кто на
            связи и что выполняется.
          </li>
        </ol>
        <div className="form-actions">
          <button className="button" type="button" onClick={onCreate}>
            Создать задачу
          </button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
