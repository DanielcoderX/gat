$proc = Start-Process -FilePath '.\bin\bench_server_conc_io.exe' -PassThru
Start-Sleep -Milliseconds 500
python tools/bench_http.py "Concurrent (10ms I/O delay)" 19894
Stop-Process -Id $proc.Id -Force
