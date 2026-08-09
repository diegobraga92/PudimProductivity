// PudimProductivity — habits load test.
//
// Exercises habit completion endpoints (the second hottest path after task CRUD):
//   BASE_URL=http://localhost:8080 k6 run infra/k6/habits-load.js
//
// Creates a habit in setup, then hammers the batch-completions read + the
// complete/uncomplete toggle under load.

import http from "k6/http";
import { check, sleep } from "k6";

const BASE = (__ENV.BASE_URL || "http://localhost:8080/api/v1").replace(/\/$/, "");
const HEADERS = {
  "Content-Type": "application/json",
  "X-User-ID": "dev-user",
  "X-User-Role": "user",
};

export const options = {
  stages: [
    { duration: "30s", target: 5 },
    { duration: "1m", target: 30 },
    { duration: "1m", target: 30 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<200"],
    http_req_failed: ["rate<0.01"],
  },
};

// Create a habit once per test run and reuse its ID.
export function setup() {
  const res = http.post(
    `${BASE}/tasks`,
    JSON.stringify({ title: "load habit", recurrence_days: ["mon", "tue", "wed", "thu", "fri"] }),
    { headers: HEADERS }
  );
  check(res, { "setup habit 201": (r) => r.status === 201 });
  return { habitId: res.json("id") };
}

export default function (data) {
  const habitId = data.habitId;

  // Each VU owns a distinct completion date so concurrent VUs don't conflict
  // (completing the same habit on the same date twice returns 409).
  const date = new Date(Date.now() - __VU * 86400000).toISOString().slice(0, 10);

  // READ: batch completions for the current week (heatmap data)
  const to = new Date().toISOString().slice(0, 10);
  const from = new Date(Date.now() - 6 * 86400000).toISOString().slice(0, 10);
  const completions = http.get(
    `${BASE}/tasks/completions?from=${from}&to=${to}`,
    { headers: HEADERS }
  );
  check(completions, { "completions 200": (r) => r.status === 200 });

  // WRITE: complete a day owned by this VU
  const complete = http.post(`${BASE}/tasks/${habitId}/complete?date=${date}`, null, { headers: HEADERS });
  check(complete, { "complete 201": (r) => r.status === 201 });

  // WRITE: uncomplete (undo)
  const uncomplete = http.del(`${BASE}/tasks/${habitId}/complete?date=${date}`, null, { headers: HEADERS });
  check(uncomplete, { "uncomplete 204": (r) => r.status === 204 });

  sleep(1);
}
