// PudimProductivity — tasks load test.
//
// Exercises the most frequent task endpoints against a running backend:
//   BASE_URL=http://localhost:8080 k6 run infra/k6/tasks-load.js
//
// Thresholds mirror the SLOs in docs/slo.md: p95 < 200ms, < 1% errors.

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
    { duration: "30s", target: 10 },  // warm-up
    { duration: "1m", target: 50 },   // ramp to 50 VUs
    { duration: "1m", target: 50 },   // hold
    { duration: "30s", target: 0 },   // ramp down
  ],
  thresholds: {
    http_req_duration: ["p(95)<200"], // task API SLO: p95 < 200ms
    http_req_failed: ["rate<0.01"],   // task API SLO: < 1% errors
  },
};

export default function () {
  // READ: default task view (unlisted one-offs)
  const list = http.get(`${BASE}/tasks?status=todo&type=one-off`, { headers: HEADERS });
  check(list, { "list 200": (r) => r.status === 200 });

  // READ: habit list
  const habits = http.get(`${BASE}/tasks?type=habit`, { headers: HEADERS });
  check(habits, { "habits 200": (r) => r.status === 200 });

  // WRITE: create a task
  const create = http.post(
    `${BASE}/tasks`,
    JSON.stringify({ title: `load-${__VU}-${__ITER}` }),
    { headers: HEADERS }
  );
  check(create, { "create 201": (r) => r.status === 201 });
  const id = create.json("id");

  // WRITE: update it
  const update = http.put(
    `${BASE}/tasks/${id}`,
    JSON.stringify({ status: "done" }),
    { headers: HEADERS }
  );
  check(update, { "update 200": (r) => r.status === 200 });

  // READ: planner view
  const scheduled = http.get(`${BASE}/tasks/scheduled`, { headers: HEADERS });
  check(scheduled, { "scheduled 200": (r) => r.status === 200 });

  sleep(1);
}
