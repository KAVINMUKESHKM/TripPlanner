import { Suspense } from "react";
import { PlannerClient } from "@/components/planner-client";

export default function PlanPage() {
  return <Suspense fallback={<main className="planner-shell"><p id="main-content" className="loading-line" tabIndex={-1}>Setting up your trip…</p></main>}><PlannerClient /></Suspense>;
}
