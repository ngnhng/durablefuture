/**
 * Copyright 2025 Nguyen Nhat Nguyen
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { type RouteConfig, index, route, layout } from "@react-router/dev/routes";

export default [
  layout("components/Layout.tsx", [
    index("routes/dashboard.tsx"),
    route("/environment", "routes/environment.tsx"),
    route("/tasks", "routes/tasks.tsx"),
    route("/runs", "routes/runs.tsx"),
    route("/batches", "routes/batches.tsx"),
    route("/schedules", "routes/schedules.tsx"),
    route("/queues", "routes/queues.tsx"),
    route("/deployments", "routes/deployments.tsx"),
    route("/test", "routes/test.tsx"),
    route("/tokens", "routes/tokens.tsx"),
    route("/bulk-actions", "routes/bulk-actions.tsx"),
    route("/api-keys", "routes/api-keys.tsx"),
    route("/environment-variables", "routes/environment-variables.tsx"),
    route("/alerts", "routes/alerts.tsx"),
    route("/preview-branches", "routes/preview-branches.tsx"),
    route("/regions", "routes/regions.tsx"),
    route("/project-settings", "routes/project-settings.tsx"),
  ])
] satisfies RouteConfig;
