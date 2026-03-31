@echo off
if "%~1"=="" (set MODEL=opusplan) else (set MODEL=%1)
dockerdoor claude --permission-mode bypassPermissions --model %MODEL%