### install
```bash
git clone https://github.com/abergasov/tcp_measurer && cd tcp_measurer
```

log collector
```bash
sudo nano /etc/fluent-bit/fluent-bit.conf
sudo systemctl restart fluent-bit
```
```bash
[INPUT]
    Name        systemd
    Tag         service_logs
    Systemd_Filter   _SYSTEMD_UNIT=tcpmeasurer.service
    Read_From_Tail   On
```