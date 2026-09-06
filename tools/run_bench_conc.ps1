$proc = Start-Process -FilePath '.\bin\bench_server_conc.exe' -PassThru
Start-Sleep -Milliseconds 500
python tools/bench_http.py Concurrent 19892
Stop-Process -Id $proc.Id -Force
