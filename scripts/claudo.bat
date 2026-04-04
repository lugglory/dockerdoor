@echo off
if "%~1"=="" (set MODEL=opus) else (set MODEL=%1)
dockerdoor claude --permission-mode bypassPermissions --model %MODEL%