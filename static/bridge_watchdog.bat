@echo off
:restart
"C:\Users\Administrator\AppData\Local\hermes\hermes-agent\venv\Scripts\python.exe" -u "D:\ai-chat\static\hermes-bridge.py"
echo %date% %time% bridge exited, restarting in 5s... >> "C:\Users\Administrator\bridge_watchdog.log"
timeout /t 5 /nobreak >nul
goto restart