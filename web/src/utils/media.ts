import type { MediaType } from "../api/library";

/** Media types in display order, shared by the Library, CSV import and Server
 *  Settings screens so their filters/selects never drift apart. */
export const MEDIA_TYPES: MediaType[] = ["movie", "series", "book", "game"];

/** i18n label key per media type. */
export const MEDIA_LABEL_KEYS: Record<MediaType, string> = {
  movie: "library.movie",
  series: "library.series",
  book: "library.book",
  game: "library.game",
};

/** Emoji used for a media type in list rows and labels. */
export const MEDIA_ICONS: Record<MediaType, string> = {
  movie: "🎬",
  series: "📺",
  book: "📚",
  game: "🎮",
};
