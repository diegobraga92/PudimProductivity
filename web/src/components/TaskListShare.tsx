import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  listTaskListMembers,
  shareTaskList,
  unshareTaskList,
  type TaskListMember,
} from "../api/collab";
import { useToast } from "./toastContext";
import { useI18n } from "../i18n";

interface TaskListShareProps {
  listId: string;
  listName: string;
  /** Whether the current user is the owner (only owners can share/revoke). */
  isOwner: boolean;
  /** Presence map listId → online user IDs (from usePresence). */
  onlineUsers?: Set<string>;
  onClose: () => void;
}

/**
 * Phase 8 share dialog: invite a user (editor/viewer), list members with live
 * presence dots, and revoke access. Owner-only for mutations.
 */
export default function TaskListShare({
  listId,
  listName,
  isOwner,
  onlineUsers,
  onClose,
}: TaskListShareProps) {
  const queryClient = useQueryClient();
  const { pushToast } = useToast();
  const { t } = useI18n();
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<"editor" | "viewer">("editor");
  const [error, setError] = useState<string | null>(null);

  const { data: members = [], isLoading } = useQuery<TaskListMember[]>({
    queryKey: ["taskListMembers", listId],
    queryFn: () => listTaskListMembers(listId),
  });

  const shareMutation = useMutation({
    mutationFn: () => shareTaskList(listId, { shared_with: userId.trim(), role }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskListMembers", listId] });
      setUserId("");
      setError(null);
      pushToast({ icon: "👥", title: t("toast.listShared"), body: t("toast.listSharedBody", { role, user: userId.trim() }) });
    },
    onError: (err: Error) => setError(err.message),
  });

  const unshareMutation = useMutation({
    mutationFn: (target: string) => unshareTaskList(listId, target),
    onSuccess: (_data, target) => {
      queryClient.invalidateQueries({ queryKey: ["taskListMembers", listId] });
      pushToast({ icon: "🔒", title: t("toast.accessRevoked"), body: t("toast.accessRevokedBody", { target }) });
    },
    onError: (err: Error) => setError(err.message),
  });

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.45)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 200,
        padding: "var(--space-md)",
      }}
      onClick={onClose}
    >
      <div
        className="card"
        style={{ maxWidth: 420, width: "100%", padding: "var(--space-lg)" }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
          <h3 style={{ margin: 0, fontSize: "var(--font-size-lg)" }}>{t("share.title", { name: listName })}</h3>
          <button className="btn btn-ghost btn-sm" onClick={onClose} aria-label={t("a11y.close")}>
            ✕
          </button>
        </div>

        {error && (
          <div className="toast toast-error" style={{ marginBottom: "var(--space-sm)" }}>
            {error}
          </div>
        )}

        {/* Invite form (owner-only) */}
        {isOwner ? (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (userId.trim()) shareMutation.mutate();
            }}
            style={{ display: "flex", gap: "0.5rem", marginBottom: "var(--space-md)" }}
          >
            <input
              className="form-input"
              placeholder={t("share.userId")}
              value={userId}
              onChange={(e) => setUserId(e.target.value)}
              style={{ flex: 1 }}
            />
            <select
              className="form-select"
              value={role}
              onChange={(e) => setRole(e.target.value as "editor" | "viewer")}
              aria-label={t("share.editor")}
            >
              <option value="editor">{t("share.editor")}</option>
              <option value="viewer">{t("share.viewer")}</option>
            </select>
            <button className="btn btn-primary" type="submit" disabled={shareMutation.isPending}>
              {t("share.invite")}
            </button>
          </form>
        ) : (
          <p className="text-sm text-secondary" style={{ marginBottom: "var(--space-md)" }}>
            {t("share.notOwner")}
          </p>
        )}

        {/* Member list */}
        {isLoading && <p className="text-sm text-secondary">{t("share.loadingMembers")}</p>}
        <div style={{ display: "flex", flexDirection: "column", gap: "0.35rem" }}>
          {members.map((member) => {
            const isOnline = onlineUsers?.has(member.shared_with) ?? false;
            return (
              <div
                key={member.shared_with}
                className="flex-center"
                style={{ gap: "0.5rem", padding: "0.35rem 0.5rem", background: "var(--color-bg-hover)", borderRadius: "8px" }}
              >
                <span
                  style={{
                    width: 8,
                    height: 8,
                    borderRadius: "50%",
                    background: isOnline ? "#00b894" : "#b2bec3",
                    flexShrink: 0,
                  }}
                  title={isOnline ? t("common.online") : t("common.offline")}
                />
                <span className="flex-1" style={{ fontSize: "var(--font-size-sm)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {member.shared_with}
                </span>
                <span className="badge" style={{ background: "var(--color-primary-light)", color: "var(--color-primary-dark)" }}>
                  {member.role}
                </span>
                {isOwner && (
                  <button
                    className="btn btn-danger btn-sm"
                    style={{ padding: "0.15rem 0.4rem", fontSize: "0.65rem" }}
                    onClick={() => unshareMutation.mutate(member.shared_with)}
                    disabled={unshareMutation.isPending}
                    title={t("share.revoke")}
                  >
                    ✕
                  </button>
                )}
              </div>
            );
          })}
          {!isLoading && members.length === 0 && (
            <p className="text-sm text-secondary">{t("share.noMembers")}</p>
          )}
        </div>
      </div>
    </div>
  );
}

