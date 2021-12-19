@echo off

SETLOCAL ENABLEEXTENSIONS
set $path=%~dp0

md dist 1>nul 2>nul

SET GOOS=windows
SET GOARCH=amd64
SET CGO_ENABLED=1
go build -ldflags "-s -w" -o "%$path%dist\%GOOS%\%GOARCH%\api.exe" "%$path%cmd\api.go"

SET GOOS=linux
SET GOARCH=amd64
SET CGO_ENABLED=0
go build -ldflags "-s -w" -o "%$path%dist\%GOOS%\%GOARCH%\api" "%$path%cmd\api.go"

SET GOOS=windows
SET GOARCH=386
SET CGO_ENABLED=1
go build -ldflags "-s -w" -o "%$path%dist\%GOOS%\%GOARCH%\api.exe" "%$path%cmd\api.go"

SET GOOS=linux
SET GOARCH=386
SET CGO_ENABLED=0
go build -ldflags "-s -w" -o "%$path%dist\%GOOS%\%GOARCH%\api" "%$path%cmd\api.go"

ENDLOCAL
