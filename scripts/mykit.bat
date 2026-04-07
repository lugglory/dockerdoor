@echo off
wt -d "%cd%" powershell -Command "gitscan"; ^
nt -d "%cd%" powershell -Command "claudo"