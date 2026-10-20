"use client";
import { useState } from "react";
import Link from "next/link";
import { Download, File, Grid2X2, History, List, Plus, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState, Fact, PageHeading } from "@/components/common";
import type { DesignAsset, MutationResult, PageData } from "@/lib/types";
import { bytes, date } from "@/lib/format";
import { DesignUpload } from "../mutations/DesignUpload";
import { FilterMenu, Pagination, useFilters } from "../query-controls";
function safeAssetURL(url: string, kind: "download" | "preview") {
  return new RegExp(`^/web/designs/[A-Za-z0-9_-]+/${kind}$`).test(url)
    ? url
    : undefined;
}
export function DesignsScreen({
  page,
  onSuccess,
}: {
  page: PageData;
  onSuccess: (result: MutationResult) => void;
}) {
  const filters = useFilters();
  const [upload, setUpload] = useState(false);
  const [uploadLocked, setUploadLocked] = useState(false);
  const [selected, setSelected] = useState<DesignAsset | null>(null);
  const [layout, setLayout] = useState("grid");
  const assets = page.designs ?? [];
  return (
    <div className="page">
      <PageHeading
        title="Дизайн"
        description={
          page.draft.design_id
            ? "Все версии выбранного материала"
            : "Материалы проекта, макеты и результаты работы"
        }
        actions={
          <button className="button" onClick={() => setUpload(true)}>
            <Plus size={14} />
            <span className="desktop-label">Загрузить</span>
          </button>
        }
      />
      <div className="toolbar">
        <div className="toolbar-start">
          <FilterMenu
            count={
              ["scope", "scope_key", "design_id"].filter((key) =>
                filters.params.has(key),
              ).length
            }
          >
            <form
              className="mutation-form"
              onSubmit={(event) => {
                event.preventDefault();
                const data = new FormData(event.currentTarget);
                filters.set({
                  scope: String(data.get("scope") ?? ""),
                  scope_key: String(data.get("scope_key") ?? ""),
                });
              }}
            >
              <label className="form-field">
                Область
                <select name="scope" defaultValue={page.draft.scope ?? ""}>
                  <option value="">Все</option>
                  <option value="project">Проект</option>
                  <option value="epic">Эпик</option>
                  <option value="task">Задача</option>
                </select>
              </label>
              <label className="form-field">
                Ключ задачи или эпика
                <input
                  name="scope_key"
                  defaultValue={page.draft.scope_key}
                  placeholder="T-1 или E-1"
                />
              </label>
              <button className="button">Применить</button>
            </form>
          </FilterMenu>
          {filters.params.size ? (
            <button
              className="button button-ghost button-sm"
              onClick={filters.reset}
            >
              Все материалы
            </button>
          ) : null}
          <span className="toolbar-count">
            {assets.length} версий в выборке
          </span>
        </div>
        <div className="segmented">
          <button
            aria-label="Сетка материалов"
            aria-pressed={layout === "grid"}
            onClick={() => setLayout("grid")}
          >
            <Grid2X2 size={16} />
          </button>
          <button
            aria-label="Список материалов"
            aria-pressed={layout === "list"}
            onClick={() => setLayout("list")}
          >
            <List size={16} />
          </button>
        </div>
      </div>
      {assets.length ? (
        <div
          className={
            layout === "grid" ? "asset-grid" : "asset-list list-surface"
          }
        >
          {assets.map((asset) => (
            <div className="asset-card" key={asset.asset_id}>
              <button
                className="asset-preview"
                onClick={() => setSelected(asset)}
                aria-label={`Просмотреть ${asset.title}, версия ${asset.version}`}
              >
                {asset.preview_url &&
                asset.media_type.startsWith("image/") &&
                safeAssetURL(asset.preview_url, "preview") ? (
                  <img
                    src={asset.preview_url}
                    alt={asset.title}
                    loading="lazy"
                  />
                ) : (
                  <File size={32} />
                )}
              </button>
              <div className="asset-body">
                <button
                  className="asset-title truncate"
                  onClick={() => setSelected(asset)}
                  title={asset.title}
                >
                  {asset.title}
                </button>
                <div className="asset-meta muted">
                  <span>
                    v{asset.version} · {bytes(asset.size)}
                  </span>
                  <span>{asset.scope_key || "Проект"}</span>
                </div>
                <div className="asset-actions">
                  <Link
                    className="button button-ghost button-xs"
                    href={`/app/designs?design_id=${asset.design_id}`}
                  >
                    <History size={12} />
                    Версии
                  </Link>
                  <a
                    className="icon-button"
                    href={safeAssetURL(asset.download_url, "download")}
                    aria-label={`Скачать ${asset.filename}`}
                  >
                    <Download size={14} />
                  </a>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          title="Материалов пока нет"
          description="Загрузите макет, изображение или документ для команды."
          action={
            <button
              className="button button-secondary"
              onClick={() => setUpload(true)}
            >
              Загрузить материал
            </button>
          }
        />
      )}
      <Pagination cursor={page.next_cursor} />
      <Dialog
        open={upload}
        onOpenChange={(open) => {
          if (!uploadLocked) setUpload(open);
        }}
      >
        <DialogContent
          showCloseButton={!uploadLocked}
          onEscapeKeyDown={(event) => {
            if (uploadLocked) event.preventDefault();
          }}
          onPointerDownOutside={(event) => {
            if (uploadLocked) event.preventDefault();
          }}
        >
          <DialogHeader>
            <DialogTitle>
              {page.draft.design_id ? "Новая версия" : "Новый материал"}
            </DialogTitle>
            <DialogDescription>
              Файл сохраняется в проекте и доступен его участникам.
            </DialogDescription>
          </DialogHeader>
          <DesignUpload
            key={page.draft.design_id ?? ""}
            page={page}
            onLockChange={setUploadLocked}
            onSuccess={(result) => {
              setUpload(false);
              onSuccess(result);
            }}
          />
        </DialogContent>
      </Dialog>
      <Dialog
        open={Boolean(selected)}
        onOpenChange={(value) => {
          if (!value) setSelected(null);
        }}
      >
        <DialogContent className="asset-dialog">
          <DialogHeader>
            <DialogTitle>{selected?.title}</DialogTitle>
            <DialogDescription>
              Версия {selected?.version} · {selected?.filename}
            </DialogDescription>
          </DialogHeader>
          {selected ? (
            <>
              <div className="asset-detail-preview">
                {selected.preview_url &&
                safeAssetURL(selected.preview_url, "preview") ? (
                  selected.media_type.startsWith("image/") ? (
                    <img src={selected.preview_url} alt={selected.title} />
                  ) : (
                    <iframe
                      src={selected.preview_url}
                      title={`Просмотр ${selected.title}`}
                      sandbox=""
                    />
                  )
                ) : (
                  <div className="compact-empty">
                    <File size={24} />
                    Предпросмотр для этого файла недоступен
                  </div>
                )}
              </div>
              <dl>
                <Fact name="Размер" value={bytes(selected.size)} />
                <Fact name="Формат" value={selected.media_type} />
                <Fact name="Загрузил" value={selected.uploader_owner} />
                <Fact name="Дата" value={date(selected.created_at)} />
                <Fact
                  name="SHA-256"
                  value={<span className="mono break">{selected.sha256}</span>}
                />
              </dl>
              <div className="form-actions">
                <Link
                  className="button button-secondary"
                  href={`/app/designs?design_id=${selected.design_id}`}
                  onClick={() => setSelected(null)}
                >
                  Все версии
                </Link>
                <a
                  className="button"
                  href={safeAssetURL(selected.download_url, "download")}
                >
                  <Download size={14} />
                  Скачать
                </a>
              </div>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
