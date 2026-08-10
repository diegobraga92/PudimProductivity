import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { addBookByISBN, addBookManual, deleteBook, listBooks, updateBookStatus } from "../api/booktrack";

const STATUSES = [
  { id: "", label: "All" },
  { id: "want_to_read", label: "Want to read" },
  { id: "reading", label: "Reading" },
  { id: "read", label: "Read" },
] as const;

export default function BookList() {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("");
  const [isbnInput, setIsbnInput] = useState("");
  const [manual, setManual] = useState<{ isbn: string; title: string } | null>(null);

  const { data: books = [], isLoading } = useQuery({
    queryKey: ["books", status],
    queryFn: () => listBooks(status || undefined),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["books"] });
  const addIsbn = useMutation({ mutationFn: (isbn: string) => addBookByISBN(isbn), onSuccess: invalidate });
  const addManual = useMutation({
    mutationFn: () => addBookManual({ isbn: manual!.isbn, title: manual!.title }),
    onSuccess: () => { invalidate(); setManual(null); },
  });
  const setBookStatus = useMutation({ mutationFn: ({ id, s }: { id: string; s: "want_to_read" | "reading" | "read" }) => updateBookStatus(id, s), onSuccess: invalidate });
  const del = useMutation({ mutationFn: (id: string) => deleteBook(id), onSuccess: invalidate });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>📚 Books</h2>
      </div>

      <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "var(--space-md)", flexWrap: "wrap" }}>
        <input
          className="input"
          placeholder="ISBN (e.g. 9781250237231)"
          value={isbnInput}
          onChange={(e) => setIsbnInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === "Enter" && isbnInput.trim()) addIsbn.mutate(isbnInput.trim()); }}
        />
        <button className="btn btn-primary" disabled={!isbnInput.trim() || addIsbn.isPending} onClick={() => addIsbn.mutate(isbnInput.trim())}>
          {addIsbn.isPending ? "Looking up…" : "Add by ISBN"}
        </button>
        <button className="btn btn-sm" onClick={() => setManual({ isbn: "", title: "" })}>Manual entry</button>
        <select className="select" value={status} onChange={(e) => setStatus(e.target.value)}>
          {STATUSES.map((s) => <option key={s.id} value={s.id}>{s.label}</option>)}
        </select>
      </div>

      {manual && (
        <div className="card" style={{ maxWidth: 480, padding: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <input className="input" style={{ width: "100%", marginBottom: "0.4rem" }} placeholder="ISBN" value={manual.isbn} onChange={(e) => setManual({ ...manual, isbn: e.target.value })} />
          <input className="input" style={{ width: "100%", marginBottom: "0.4rem" }} placeholder="Title" value={manual.title} onChange={(e) => setManual({ ...manual, title: e.target.value })} />
          <button className="btn btn-primary btn-sm" disabled={!manual.isbn || !manual.title} onClick={() => addManual.mutate()}>Save</button>
          <button className="btn btn-ghost btn-sm" onClick={() => setManual(null)}>Cancel</button>
        </div>
      )}

      {addIsbn.error && <p style={{ color: "var(--color-danger)" }}>{String(addIsbn.error)}</p>}
      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Loading books…</p>}
      {books.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">📚</div>
          <p className="empty-state-text">No books yet. Add one by ISBN above!</p>
        </div>
      )}

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))", gap: "var(--space-md)" }}>
        {books.map((b) => (
          <div key={b.id} className="card">
            {b.thumbnail_url ? (
              <img src={b.thumbnail_url} alt={b.title} style={{ width: "100%", height: 180, objectFit: "cover", borderRadius: "var(--radius-md)" }} />
            ) : (
              <div style={{ height: 180, display: "flex", alignItems: "center", justifyContent: "center", background: "var(--color-bg-muted)", borderRadius: "var(--radius-md)" }}>📖</div>
            )}
            <p className="card-title" style={{ margin: "0.4rem 0 0.2rem", fontSize: "var(--font-size-sm)" }}>{b.title}</p>
            <p style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              {b.authors?.join(", ") || "Unknown author"} · {b.page_count} pp
            </p>
            <select
              className="select" style={{ marginTop: "0.35rem", width: "100%" }}
              value={b.status}
              onChange={(e) => setBookStatus.mutate({ id: b.id, s: e.target.value as "want_to_read" | "reading" | "read" })}
            >
              <option value="want_to_read">Want to read</option>
              <option value="reading">Reading</option>
              <option value="read">Read</option>
            </select>
            <button className="btn btn-danger btn-sm" style={{ marginTop: "0.35rem" }} onClick={() => del.mutate(b.id)}>Delete</button>
          </div>
        ))}
      </div>
    </div>
  );
}
