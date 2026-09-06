$proc = Start-Process -FilePath '.\bin\bench_server_seq.exe' -PassThru
Start-Sleep -Milliseconds 500
python tools/bench_http.py Sequential 19891
Stop-Process -Id $proc.Id -Force
