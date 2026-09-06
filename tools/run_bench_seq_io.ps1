$proc = Start-Process -FilePath '.\bin\bench_server_seq_io.exe' -PassThru
Start-Sleep -Milliseconds 500
python tools/bench_http.py "Sequential (10ms I/O delay)" 19893
Stop-Process -Id $proc.Id -Force
