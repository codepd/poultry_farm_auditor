# Repository Engineering Cadence & Rules

## 1. Test-With-Every-Fix Cadence
Whenever a bug fix or behavior change is implemented in backend (`go_api/`) or frontend (`react_frontend/`):
- **Go Backend**: Add or update unit/handler tests in `go_api/handlers/*_test.go` and run `cd go_api && go test -v ./handlers ./middleware`.
- **Functional API E2E**: Add or update live API assertions in `scripts/run-functional-e2e-tests.py` (executed in PR CI against Postgres + Go API).
- **React Frontend**: Add or update Jest / React Testing Library tests (`react_frontend/src/**/*.test.tsx`) and run `cd react_frontend && npm test -- --watchAll=false --ci`.

## 2. Git / PR Hygiene
- Always check `gh pr view <num> --json state` before pushing follow-up commits to an existing PR branch. If the PR is already `MERGED` or `CLOSED`, create a new branch from `origin/main` and open a new PR.
