import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/booktrack-v1";

// Types are generated from api/openapi/booktrack-v1.yaml.
export type Book = components["schemas"]["Book"];
export type BookStatus = components["schemas"]["BookStatus"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

export async function listBooks(status?: string): Promise<Book[]> {
  const q = status ? `?status=${encodeURIComponent(status)}` : "";
  const res = await fetch(`${config.apiBaseUrl}/books${q}`);
  if (!res.ok) await handleError(res, `Failed to list books: ${res.status}`);
  return res.json() as Promise<Book[]>;
}

export async function addBookByISBN(isbn: string): Promise<Book> {
  const res = await fetch(`${config.apiBaseUrl}/books/by-isbn`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify({ isbn }),
  });
  if (!res.ok) await handleError(res, `Failed to add book: ${res.status}`);
  return res.json() as Promise<Book>;
}

export async function addBookManual(req: {
  isbn: string;
  title: string;
  authors?: string[];
}): Promise<Book> {
  const res = await fetch(`${config.apiBaseUrl}/books`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to add book: ${res.status}`);
  return res.json() as Promise<Book>;
}

export async function updateBookStatus(bookId: string, status: BookStatus): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/books/${bookId}/status`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify({ status }),
  });
  if (!res.ok) await handleError(res, `Failed to update status: ${res.status}`);
}

export async function deleteBook(bookId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/books/${bookId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete book: ${res.status}`);
}
