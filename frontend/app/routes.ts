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
