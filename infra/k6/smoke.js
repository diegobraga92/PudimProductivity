// PudimProductivity — CI smoke load test.
//
// A short, low-VU pass over the hottest task + habit endpoints. Runs in CI on
// every backend push (see .github/workflows/backend-ci.yml → load-smoke job)
// to catch SLO regressions early without the cost of the full load suite
// (infra/k6/tasks-load.js, infra/k6/habits-load.js).
//
//   BASE_URL=http://localhost:8080/api/v1 k6 run infra/k6/smoke.js
//
// Thresholds mirror docs/slo.md: p95 < 200ms, < 1% errors.

import http from "k6/http";
import { check, sleep } from "k6";

const BASE = (__ENV.BASE_URL || "http://localhost:8080/api/v1").replace(/\/$/, "");
const HEADERS = {
  "Content-Type": "application/json",
  "X-User-ID": "dev-user",
  "X-User-Role": "user",
};

export const options = {
  vus: 5,
  duration: "20s",
  thresholds: {
    http_req_duration: ["p(95)<200"], // task API SLO: p95 < 200ms
    http_req_failed: ["rate<0.01"],   // task API SLO: < 1% errors
    checks: ["rate>0.95"],
  },
};

export function setup() {
  const res = http.post(
    `${BASE}/tasks`,
    JSON.stringify({ title: "smoke habit", recurrence_days: ["mon", "wed", "fri"] }),
    { headers: HEADERS }
  );
  check(res, { "setup habit 201": (r) => r.status === 201 });
  return { habitId: res.json("id") };
}

export default function (data) {
  const habitId = data.habitId;

  // READ: default task view + habit list
  const list = http.get(`${BASE}/tasks?status=todo&type=one-off`, { headers: HEADERS });
  check(list, { "list 200": (r) => r.status === 200 });

  const habits = http.get(`${BASE}/tasks?type=habit`, { headers: HEADERS });
  check(habits, { "habits 200": (r) => r.status === 200 });

  // READ: planner view
  const scheduled = http.get(`${BASE}/tasks/scheduled`, { headers: HEADERS });
  check(scheduled, { "scheduled 200": (r) => r.status === 200 });

  // READ: batch completions (heatmap data)
  const to = new Date().toISOString().slice(0, 10);
  const from = new Date(Date.now() - 6 * 86400000).toISOString().slice(0, 10);
  const completions = http.get(`${BASE}/tasks/completions?from=${from}&to=${to}`, { headers: HEADERS });
  check(completions, { "completions 200": (r) => r.status === 200 });

  // WRITE: create + update a one-off task
  const create = http.post(`${BASE}/tasks`, JSON.stringify({ title: `smoke-${__VU}-${__ITER}` }), { headers: HEADERS });
  check(create, { "create 201": (r) => r.status === 201 });
  const id = create.json("id");

  const update = http.put(`${BASE}/tasks/${id}`, JSON.stringify({ status: "done" }), { headers: HEADERS });
  check(update, { "update 200": (r) => r.status === 200 });

  // WRITE: habit completion toggle (iteration-unique date avoids 409 conflicts).
  // Each (VU, iteration) pair maps to a unique day index via the bijection
  // ITER*5 + (VU-1): 5 VUs → no two requests ever touch the same date, so a
  // per-VU date (as in habits-load.js) would collide across iterations.
  const iterOffset = (__ITER * 5) + (__VU - 1);
  const date = new Date(Date.now() - iterOffset * 86400000).toISOString().slice(0, 10);
  const complete = http.post(`${BASE}/tasks/${habitId}/complete?date=${date}`, null, { headers: HEADERS });
  check(complete, { "complete 201": (r) => r.status === 201 });

  const uncomplete = http.del(`${BASE}/tasks/${habitId}/complete?date=${date}`, null, { headers: HEADERS });
  check(uncomplete, { "uncomplete 204": (r) => r.status === 204 });

  sleep(0.5);
}
