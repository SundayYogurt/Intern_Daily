$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$serverDir = Join-Path $projectRoot 'server'
$exe = Join-Path $serverDir 'internship-journal.exe'
if (-not (Test-Path -LiteralPath $exe)) {
  throw 'ไม่พบไฟล์ server/internship-journal.exe กรุณาติดตั้ง Go แล้วรัน go run . ในโฟลเดอร์ server'
}
Set-Location -LiteralPath $serverDir
Write-Host 'เปิด http://127.0.0.1:8080 ในเบราว์เซอร์'
& $exe
