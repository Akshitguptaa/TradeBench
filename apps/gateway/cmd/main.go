package main

// =============================================================================
// TradeBench API Gateway — Route Contract
// =============================================================================
//
// All routes are served on :8080. JWT (Authorization: Bearer <token>) is
// required for every route except /health.
//
//   POST   /api/v1/auth/token          → issue JWT (dev mode: no real auth)
//   POST   /api/v1/submissions         → upload binary → submission svc
//   GET    /api/v1/submissions/:id     → get submission status
//   GET    /api/v1/runs/:id            → get run status + metrics
//   GET    /api/v1/leaderboard         → current top-50 (REST fallback)
//   WS     /ws/leaderboard             → WebSocket stream → leaderboard svc
//   GET    /health                     → liveness probe (no auth)
//
// See docs/api-contract.md for full request/response schemas.
// =============================================================================

import "fmt"

func main() {
	fmt.Println("gateway service started")
	select {}
}
