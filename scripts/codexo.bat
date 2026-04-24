@echo off
if "%~1"=="" (set MODEL_ARG=) else (set MODEL_ARG=--model %1)
dockerdoor codex --dangerously-bypass-approvals-and-sandbox %MODEL_ARG%
