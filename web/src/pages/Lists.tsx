import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useConfirm } from "../components/useConfirm";
import {
  listTaskLists,
  createTaskList,
  deleteTaskList,
  type TaskList,
} from "../api/taskLists";
import ListDetailPanel from "../components/ListDetailPanel";
import TaskListShare from "../components/TaskListShare";
import QuickAddForm from "../components/QuickAddForm";
import { usePresence } from "../hooks/usePresence";
import { useI18n } from "../i18n";
import { DEV_USER_ID } from "../api/client";

export default function Lists() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [selectedListForDetail, setSelectedListForDetail] = useState<string | null>(null);
  const [newListName, setNewListName] = useState("");
  // Which list's share dialog is open.
  const [shareListId, setShareListId] = useState<string | null>(null);

  // Fetch task lists
  const { data: taskLists = [] } = useQuery<TaskList[]>({
    queryKey: ["taskLists"],
    queryFn: listTaskLists,
  });

  // Live presence — who is online per list.
  const { online: onlineByList } = usePresence(taskLists.map((l) => l.id));

  const confirm = useConfirm();

  const createListMutation = useMutation({
    mutationFn: (name: string) => createTaskList({ name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskLists"] });
      setNewListName("");
    },
  });

  const deleteListMutation = useMutation({
    mutationFn: deleteTaskList,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskLists"] });
      setSelectedListForDetail(null);
    },
  });

  return (
    <div className="animate-fade-in">
      <div className="page-header">
        <div>
          <h1 className="page-heading" style={{ marginBottom: "0.25rem" }}>
            {t("lists.title")}
          </h1>
          <p className="page-subtitle">
            {t("lists.subtitle")}
          </p>
        </div>
      </div>

      <div className="lists-inner">
        {/* ===== LEFT: LIST PICKER ===== */}
        <div className="list-picker">
          {/* Create new list */}
          <QuickAddForm
            value={newListName}
            onChange={setNewListName}
            onSubmit={() => {
              if (newListName.trim()) createListMutation.mutate(newListName.trim());
            }}
            placeholder={t("lists.newName")}
            submitLabel={t("lists.create")}
            isPending={createListMutation.isPending}
          />

          {taskLists.length === 0 && (
            <div className="empty-state" style={{ padding: "var(--space-md)" }}>
              <div className="empty-state-icon" style={{ fontSize: "1.5rem" }}>📁</div>
              <p className="empty-state-text">{t("lists.empty")}</p>
            </div>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: "0.35rem" }}>
            {taskLists.map((list) => {
              const ownerId = list.owner_id ?? "";
              const isOwner = ownerId === DEV_USER_ID;
              const ownerOnline = onlineByList.get(list.id)?.has(ownerId) ?? false;
              return (
                <div
                  key={list.id}
                  className={`list-picker-item ${selectedListForDetail === list.id ? "selected" : ""}`}
                  onClick={() => setSelectedListForDetail(list.id)}
                >
                  <span style={{ fontSize: "1rem" }}>📁</span>
                  {/* Owner presence dot */}
                  <span
                    style={{
                      width: 8,
                      height: 8,
                      borderRadius: "50%",
                      background: ownerOnline ? "#00b894" : "#b2bec3",
                      flexShrink: 0,
                    }}
                    title={t("lists.ownerStatus", { id: list.owner_id ?? "", status: ownerOnline ? t("common.online") : t("common.offline") })}
                  />
                  <span
                    className="flex-1"
                    style={{
                      fontWeight: selectedListForDetail === list.id ? 600 : 500,
                      fontSize: "var(--font-size-sm)",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {list.name}
                  </span>
                  {/* Share dialog */}
                  <button
                    className="btn btn-ghost btn-sm"
                    style={{ padding: "0.15rem 0.4rem", fontSize: "0.7rem" }}
                    title={t("lists.share")}
                    onClick={(e) => {
                      e.stopPropagation();
                      setShareListId(list.id);
                    }}
                  >
                    👥
                  </button>
                  {/* Only the owner can delete a list */}
                  {isOwner && (
                    <button
                      className="btn btn-danger btn-sm"
                      style={{ padding: "0.15rem 0.4rem", fontSize: "0.65rem" }}
                      onClick={async (e) => {
                        e.stopPropagation();
                        const ok = await confirm({
                          title: t("lists.deleteConfirm", { name: list.name }),
                          confirmLabel: t("tasks.delete"),
                          confirmVariant: "danger",
                        });
                        if (ok) {
                          deleteListMutation.mutate(list.id);
                        }
                      }}
                    >
                      ✕
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* ===== RIGHT: LIST DETAIL ===== */}
        <div className="list-detail">
          {selectedListForDetail ? (
            <ListDetailPanel
              listId={selectedListForDetail}
              onListDeleted={() => {
                queryClient.invalidateQueries({ queryKey: ["taskLists"] });
                setSelectedListForDetail(null);
              }}
            />
          ) : (
            <div className="empty-state">
              <div className="empty-state-icon">👈</div>
              <p className="empty-state-text">{t("lists.selectPrompt")}</p>
            </div>
          )}
        </div>
      </div>

      {/* Share dialog */}
      {shareListId &&
        (() => {
          const list = taskLists.find((l) => l.id === shareListId);
          if (!list) return null;
          return (
            <TaskListShare
              listId={list.id}
              listName={list.name}
              isOwner={list.owner_id === DEV_USER_ID}
              onlineUsers={onlineByList.get(list.id)}
              onClose={() => setShareListId(null)}
            />
          );
        })()}
    </div>
  );
}

