@echo off
echo Running pivot via go (CGO required for SQLite)...
set CGO_ENABLED=1
go run ./cmd/pivot %*
