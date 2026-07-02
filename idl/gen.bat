@echo off
echo ============================================
echo           Generating all protocols
echo ============================================

call gen\client.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\user_server.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\homepage_server.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\gateway.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\match_server.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\server_manager.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\item_manager.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\auction.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\route.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\route_test.bat
if %errorlevel% neq 0 exit /b %errorlevel%

call gen\ranking.bat
if %errorlevel% neq 0 exit /b %errorlevel%

echo ============================================
echo           All protocols generated
echo ============================================